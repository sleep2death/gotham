package gotham

import (
	"bufio"
	"errors"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ErrAbortHandler is a sentinel panic value to abort a handler.
var ErrAbortHandler = errors.New("net/tcp: abort Handler")

// A ConnState represents the state of a client connection to a server.
type ConnState int

const (
	hdrLen = 4
	// StateNew represents a new connection that is expected to
	StateNew ConnState = iota

	// StateActive represents a connection that has read 1 or more
	StateActive

	// StateIdle represents a connection that has finished
	StateIdle

	// StateClosed represents a closed connection.
	StateClosed
)

type conn struct {
	// server is the server on which the connection arrived.
	// Immutable; never nil.
	server *Server

	// rwc is the underlying network connection.
	// This is never wrapped by other types and is the value given out
	// to CloseNotifier callers. It is usually of type *net.TCPConn or
	// *tls.Conn.
	rwc net.Conn

	// remoteAddr is rwc.RemoteAddr().String(). It is not populated synchronously
	// inside the Listener's Accept goroutine, as some implementations block.
	// It is populated immediately inside the (*conn).serve goroutine.
	// This is the value of a Handler's (*Request).RemoteAddr.
	remoteAddr string

	// werr is set to the first write error to rwc.
	// It is set via checkConnErrorWriter{w}, where bufw writes.
	werr error

	// bufr reads from r.
	bufr *bufio.Reader

	// bufw writes to checkConnErrorWriter{c}, which populates werr on error.
	bufw *bufio.Writer

	curState struct{ atomic uint64 } // packed (unixtime<<8|uint8(ConnState))

	// mu guards hijackedv
	mu sync.Mutex
}

var stateName = map[ConnState]string{
	StateNew:    "new",
	StateActive: "active",
	StateIdle:   "idle",
	StateClosed: "closed",
}

func (c ConnState) String() string {
	return stateName[c]
}

func (c *conn) setState(nc net.Conn, state ConnState) {
	srv := c.server
	switch state {
	case StateNew:
		srv.trackConn(c, true)
	case StateClosed:
		srv.trackConn(c, false)
	}
	if state > 0xff || state < 0 {
		panic("internal error")
	}
	packedState := uint64(time.Now().Unix()<<8) | uint64(state)
	atomic.StoreUint64(&c.curState.atomic, packedState)
	if hook := srv.ConnState; hook != nil {
		hook(nc, state)
	}
}

func (c *conn) getState() (state ConnState, unixSec int64) {
	packedState := atomic.LoadUint64(&c.curState.atomic)
	return ConnState(packedState & 0xff), int64(packedState >> 8)
}

func (c *conn) finalFlush() {
	if c.bufr != nil {
		// Steal the bufio.Reader (~4KB worth of memory) and its associated
		// reader for a future connection.
		putBufioReader(c.bufr)
		c.bufr = nil
	}

	if c.bufw != nil {
		if err := c.bufw.Flush(); err != nil {
			panic(err)
		}
		// Steal the bufio.Writer (~4KB worth of memory) and its associated
		// writer for a future connection.
		putBufioWriter(c.bufw)
		c.bufw = nil
	}
}

// Close the connection.
func (c *conn) close() {
	c.finalFlush()
	if err := c.rwc.Close(); err != nil {
		c.server.logf(err.Error())
	}
}

// Serve a new connection.
func (c *conn) serve() {
	// set remote addr
	c.remoteAddr = c.rwc.RemoteAddr().String()

	defer func() {
		// recover from reading panic, if failed log the err
		if err := recover(); err != nil && err != ErrAbortHandler {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("tcp: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		// close the connection
		// it will flush the writer, and put the reader&writer back to pool
		c.close()
		// untrack the connection
		c.setState(c.rwc, StateClosed)
	}()

	// wrap the underline conn with bufio reader&writer
	// sync pool inside
	c.bufr = newBufioReader(c.rwc)
	c.bufw = newBufioWriter(c.rwc)

	rErr := func(err error) {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			panic(err)
		} else if err != nil {
			panic("unexpected reading error:" + err.Error())
		}
	}

	for {
		fh, err := ReadFrameHeader(c.bufr)
		rErr(err)

		// set underline conn to active mode
		c.setState(c.rwc, StateActive)

		if fh.Length == 0 {
			// not enough data for future reading
			continue
		}

		fb := make([]byte, fh.Length)

		_, err = io.ReadFull(c.bufr, fb)
		rErr(err)

		// TODO: message handler here
		// c.server.logf("msg<%s", fb)
		c.server.ServeTCP(c.bufw, fh, fb)

		if d := c.server.idleTimeout(); d != 0 {
			_ = c.rwc.SetReadDeadline(time.Now().Add(d))
		} else {
			_ = c.rwc.SetReadDeadline(time.Time{})
		}

		// set underline conn back to idle mode
		c.setState(c.rwc, StateIdle)
	}
}

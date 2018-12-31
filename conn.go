package gotham

import (
	"bufio"
	"encoding/binary"
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
		c.bufw.Flush()
		// Steal the bufio.Writer (~4KB worth of memory) and its associated
		// writer for a future connection.
		putBufioWriter(c.bufw)
		c.bufw = nil
	}
}

// Close the connection.
func (c *conn) close() {
	c.finalFlush()
	c.rwc.Close()
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
		c.close()
		// untrack the connection
		c.setState(c.rwc, StateClosed)
	}()

	c.bufr = newBufioReader(c.rwc)
	c.bufw = newBufioWriter(c.rwc)

	header := make([]byte, 4)

	for {
		_, err := io.ReadFull(c.bufr, header)
		if err != nil {
			panic("header reading error: " + err.Error())
		}

		size := binary.BigEndian.Uint32(header)

		if size == 0 {
			c.server.logf("idle")
			c.setState(c.rwc, StateIdle)
			continue
		}

		msg := make([]byte, size)
		_, err = io.ReadFull(c.bufr, msg)

		if err != nil {
			panic("body reading error: " + err.Error())
		}

		c.setState(c.rwc, StateActive)
		c.server.logf("active: %s", msg)

		if d := c.server.IdleTimeout; d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
			if _, err := c.bufr.Peek(4); err != nil {
				c.server.logf("peek error: %s", err.Error())
				continue
			}
		}

		c.rwc.SetReadDeadline(time.Time{})
	}
}

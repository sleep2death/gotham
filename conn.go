package gotham

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

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
	// to CloseNotifier callers. It is usually of type *net.TCPConn
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
		// flush it, anyway
		_ = c.bufw.Flush()
		// Steal the bufio.Writer (~4KB worth of memory) and its associated
		// writer for a future connection.
		putBufioWriter(c.bufw)
		c.bufw = nil
	}
}

// Close the connection.
func (c *conn) close() {
	c.finalFlush()
	_ = c.rwc.Close()
}

// Serve a new connection.
func (c *conn) serve() {
	// set remote addr
	c.remoteAddr = c.rwc.RemoteAddr().String()

	defer func() {
		// recover from reading panic, if failed log the err
		if err := recover(); err != nil && c.server.shuttingDown() == false {
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

	// conn loop start
	for {
		fh, err := ReadFrameHeader(c.bufr)

		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			panic(err)
		}

		// set underline conn to active mode
		c.setState(c.rwc, StateActive)

		// if fh.Length == 0 {
		// 	// not enough data for future reading
		// 	continue
		// }

		fb := make([]byte, fh.Length)

		_, err = io.ReadFull(c.bufr, fb)

		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			panic(err)
		}

		// TODO: message handler here
		// c.server.logf("msg<%s", fb)
		if c.server.ServeTCP != nil {
			c.server.ServeTCP(c.bufw, fh, fb)
		}

		if d := c.server.idleTimeout(); d != 0 {
			err = c.rwc.SetReadDeadline(time.Now().Add(d))
		} else {
			err = c.rwc.SetReadDeadline(time.Time{})
		}

		// set underline conn back to idle mode
		c.setState(c.rwc, StateIdle)

		if err != nil {
			panic(err)
		}
	}
}

// FRAME -------------------------------------------------

// A FrameType is a registered frame type as defined in
// http://http2.github.io/http2-spec/#rfc.section.11.2
type FrameType uint8

const frameHeaderLen = 5

const (
	// FrameData type
	FrameData FrameType = 0x0
	// FrameSettings type
	FrameSettings FrameType = 0x1
	// FramePing type
	FramePing FrameType = 0x2
)

var frameName = map[FrameType]string{
	FrameData:     "DATA",
	FrameSettings: "SETTINGS",
	FramePing:     "PING",
}

func (t FrameType) String() string {
	if s, ok := frameName[t]; ok {
		return s
	}
	return fmt.Sprintf("UNKNOWN_FRAME_TYPE_%d", uint8(t))
}

const (
	minMaxFrameSize = 1 << 14
	maxFrameSize    = 1<<24 - 1
)

// Flags is a bitmask of HTTP/2 flags.
// The meaning of flags varies depending on the frame type.
type Flags uint8

// Has reports whether f contains all (0 or more) flags in v.
func (f Flags) Has(v Flags) bool {
	return (f & v) == v
}

// Frame-specific FrameHeader flag bits.
const (
	// check flag for validating the frame
	FlagFrameAck Flags = 0x10

	// Data Frame
	// FlagDataEndStream Flags = 0x10

	// Settings Frame
	// FlagSettingsAck Flags = 0x10

	// Ping Frame
	// FlagPingAck Flags = 0x10
)

// ErrFrameTooLarge is returned from Framer.ReadFrame when the peer
// sends a frame that is larger than declared with SetMaxReadFrameSize.
var ErrFrameTooLarge = errors.New("tcp: frame too large")

// ErrFrameFlags is returned from ReadFrame when Flags.has returned false
var ErrFrameFlags = errors.New("tcp: frame flags error")

var logReads, logWrites bool

// FrameHeader store the reading data header
type FrameHeader struct {
	// Type is the 1 byte frame type.
	Type FrameType
	// Flags are the 1 byte of 8 potential bit flags per frame.
	// They are specific to the frame type.
	Flags Flags
	// Length is the length of the frame, not including the 9 byte header.
	// The maximum size is one byte less than 16MB (uint24), but only
	// frames up to 16KB are allowed without peer agreement.
	Length uint32
}

func (fh *FrameHeader) validate() error {
	// frame body size check
	if fh.Length > maxFrameSize {
		return ErrFrameTooLarge
	}

	// frameack flag check for validating the data
	if fh.Flags.Has(FlagFrameAck) == false {
		return ErrFrameFlags
	}

	// TODO: specific frame type check
	return nil
}

// WriteData writes a data frame.
func WriteData(w io.Writer, data []byte) (err error) {
	var flags Flags
	// flags |= FlagDataEndStream
	flags |= FlagFrameAck

	length := len(data)
	if length >= (1 << 24) {
		return ErrFrameTooLarge
	}

	header := [frameHeaderLen]byte{
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
		byte(FrameData),
		byte(flags),
	}
	wbuf := append(header[:frameHeaderLen], data...)

	n, err := w.Write(wbuf)

	if err == nil && n != len(wbuf) {
		err = io.ErrShortWrite
	} else if err != nil {
		return
	}

	return err
}

// ReadFrameHeader from the given io reader
func ReadFrameHeader(r io.Reader) (FrameHeader, error) {
	pbuf := fhBytes.Get().(*[]byte)
	defer fhBytes.Put(pbuf)

	buf := *(pbuf)

	_, err := io.ReadFull(r, buf[:frameHeaderLen])

	if err != nil {
		return FrameHeader{}, err
	}

	fh := FrameHeader{
		Length: (uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])),
		Type:   FrameType(buf[3]),
		Flags:  Flags(buf[4]),
	}

	err = fh.validate()
	return fh, err
}

// frame header bytes pool.
// Used only by ReadFrameHeader.
var fhBytes = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, frameHeaderLen)
		return &buf
	},
}

// POOL  -------------------------------------------------

var (
	bufioReaderPool sync.Pool
	bufioWriterPool sync.Pool
)

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewReader(r)
}

func putBufioReader(br *bufio.Reader) {
	br.Reset(nil)
	bufioReaderPool.Put(br)
}

func newBufioWriter(w io.Writer) *bufio.Writer {
	if v := bufioWriterPool.Get(); v != nil {
		bw := v.(*bufio.Writer)
		bw.Reset(w)
		return bw
	}
	// Note: if this reader size is ever changed, update
	// TestHandlerBodyClose's assumptions.
	return bufio.NewWriter(w)
}

func putBufioWriter(bw *bufio.Writer) {
	bw.Reset(nil)
	bufioWriterPool.Put(bw)
}

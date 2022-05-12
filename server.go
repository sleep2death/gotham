package gotham

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ErrServerClosed is returned by the Server's Serve,
var ErrServerClosed = errors.New("tcp: Server closed")

type Handler interface {
	ServeProto(ResponseWriter, *Request)
}

type Codec interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, req *Request) error
}

// Server instance
type Server struct {
	// Addr optionally specifies the TCP address for the server to listen on,
	Addr string

	Handler Handler // handler to invoke

	Codec Codec // encoding and decoding data

	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	ReadTimeout time.Duration
	// WriteTimeout is the maximum duration before timing out
	// writes of the response.
	WriteTimeout time.Duration
	// IdleTimeout is the maximum amount of time to wait for the
	// next request.
	IdleTimeout time.Duration

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state.
	ConnState func(net.Conn, ConnState)

	// ErrorLog specifies an optional logger for errors accepting
	// connections, unexpected behavior from handlers, and
	// underlying FileSystem errors.
	ErrorLog *log.Logger

	inShutdown int32 // accessed atomically (non-zero means we're in Shutdown)

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{}
	activeConn map[*conn]struct{}
	doneChan   chan struct{}
	onShutdown []func()
}

func ListenAndServe(addr string, handler Handler, codec Codec) error {
	server := &Server{Addr: addr, Handler: handler, Codec: codec}
	return server.ListenAndServe()
}

func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() {
		return ErrServerClosed
	}

	addr := srv.Addr
	if len(addr) == 0 {
		return errors.New("empty address")
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

// Serve the given listener
func (srv *Server) Serve(l net.Listener) error {
	l = &onceCloseListener{Listener: l}

	defer func() {
		if err := l.Close(); err != nil {
			panic(err)
		}
	}()

	// serve multiple listeners
	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, e := l.Accept()
		if e != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("tcp: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c := srv.newConn(rw)
		c.setState(c.rwc, StateNew) // before Serve can return
		// do not need context, 'cause the connect is going to connect forever
		go c.serve()
	}
}

// onceCloseListener wraps a net.Listener, protecting it from
// multiple Close calls.
type onceCloseListener struct {
	net.Listener
	once     sync.Once
	closeErr error
}

func (oc *onceCloseListener) Close() error {
	oc.once.Do(oc.close)
	return oc.closeErr
}

func (oc *onceCloseListener) close() { oc.closeErr = oc.Listener.Close() }

func (srv *Server) getDoneChan() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.getDoneChanLocked()
}

func (srv *Server) getDoneChanLocked() chan struct{} {
	if srv.doneChan == nil {
		srv.doneChan = make(chan struct{})
	}
	return srv.doneChan
}

func (srv *Server) closeDoneChanLocked() {
	ch := srv.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(ch)
	}
}

func (srv *Server) logf(format string, args ...interface{}) {
	if srv.ErrorLog != nil {
		srv.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

// Create new connection from rwc.
func (srv *Server) newConn(rwc net.Conn) *conn {
	c := &conn{
		server: srv,
		rwc:    rwc,
	}
	return c
}

func (srv *Server) shuttingDown() bool {
	// TODO: replace inShutdown with the existing atomicBool type;
	// see https://github.com/golang/go/issues/20239#issuecomment-381434582
	return atomic.LoadInt32(&srv.inShutdown) != 0
}

// trackListener adds or removes a net.Listener to the set of tracked
// listeners.
//
// We store a pointer to interface in the map set, in case the
// net.Listener is not comparable. This is safe because we only call
// trackListener via Serve and can track+defer untrack the same
// pointer to local variable there. We never need to compare a
// Listener from another caller.
//
// It reports whether the server is still up (not Shutdown or Closed).
func (srv *Server) trackListener(ln *net.Listener, add bool) bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.listeners == nil {
		srv.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if srv.shuttingDown() {
			return false
		}
		srv.listeners[ln] = struct{}{}
	} else {
		delete(srv.listeners, ln)
	}
	return true
}

func (srv *Server) trackConn(c *conn, add bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.activeConn == nil {
		srv.activeConn = make(map[*conn]struct{})
	}
	if add {
		srv.activeConn[c] = struct{}{}
	} else {
		delete(srv.activeConn, c)
	}
}

// Close immediately closes all active net.Listeners and any
// connections in state StateNew, StateActive, or StateIdle. For a
// graceful shutdown, use Shutdown.
//
// Close does not attempt to close (and does not even know about)
// any hijacked connections, such as WebSockets.
//
// Close returns any error returned from closing the Server's
// underlying Listener(s).
func (srv *Server) Close() error {
	atomic.StoreInt32(&srv.inShutdown, 1)
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.closeDoneChanLocked()
	err := srv.closeListenersLocked()
	for c := range srv.activeConn {
		_ = c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return err
}

func (srv *Server) closeListenersLocked() error {
	var err error
	for ln := range srv.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
		delete(srv.listeners, ln)
	}
	return err
}

// shutdownPollInterval is how often we poll for quiescence
// during Server.Shutdown. This is lower during tests, to
// speed up tests.
// Ideally we could find a solution that doesn't involve polling,
// but which also doesn't have a high runtime cost (and doesn't
// involve any contentious mutexes), but that is left as an
// exercise for the reader.
var shutdownPollInterval = 500 * time.Millisecond

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing all open
// listeners, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener(s).
//
// When Shutdown is called, Serve, ListenAndServe, and
// ListenAndServeTLS immediately return ErrServerClosed. Make sure the
// program doesn't exit and waits instead for Shutdown to return.
//
// Shutdown does not attempt to close nor wait for hijacked
// connections such as WebSockets. The caller of Shutdown should
// separately notify such long-lived connections of shutdown and wait
// for them to close, if desired. See RegisterOnShutdown for a way to
// register shutdown notification functions.
//
// Once Shutdown has been called on a server, it may not be reused;
// future calls to methods such as Serve will return ErrServerClosed.
func (srv *Server) Shutdown() error {
	srv.logf("Start to shutdown...")

	atomic.StoreInt32(&srv.inShutdown, 1)

	srv.mu.Lock()
	lnerr := srv.closeListenersLocked()
	srv.closeDoneChanLocked()
	for _, f := range srv.onShutdown {
		go f()
	}
	srv.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if srv.closeIdleConns() {
			srv.logf("Shutdown completed")
			return lnerr
		}
		select {
		case <-ticker.C:
			srv.mu.Lock()
			num := len(srv.activeConn)
			srv.mu.Unlock()
			srv.logf("waiting on %v connections", num)
		}
	}
}

// closeIdleConns closes all idle connections and reports whether the
// server is quiescent.
func (srv *Server) closeIdleConns() bool {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	quiescent := true
	for c := range srv.activeConn {
		st, unixSec := c.getState()
		// treat StateNew connections as if
		// they're idle if we haven't read the first request's
		// header in over 5 seconds.
		if st == StateNew && unixSec < time.Now().Unix()-5 {
			st = StateIdle
		}
		if st != StateIdle || unixSec == 0 {
			// Assume unixSec == 0 means it's a very new
			// connection, without state set yet.
			quiescent = false
			continue
		}

		_ = c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return quiescent
}

func (srv *Server) idleTimeout() time.Duration {
	if srv.IdleTimeout != 0 {
		return srv.IdleTimeout
	}
	return srv.ReadTimeout
}

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

// Request wrap the connection and other userful information of the client's request
type Request struct {
	conn    *conn
	TypeURL string
	Data    []byte
}

func (req *Request) RemoteAddr() string {
	if req.conn != nil {
		return req.conn.remoteAddr
	}
	return "0.0.0.0"
}

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
	// close it anyway
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
		// handle connection timeout
		if d := c.server.ReadTimeout; d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
		}
		// read frame header
		fh, err := ReadFrameHeader(c.bufr)
		// log.Print(fh)
		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			// TODO: log error instead?
			panic(err)
		}

		// set underline conn to active mode
		c.setState(c.rwc, StateActive)

		if fh.Length > 0 {
			req, err := ReadFrameBody(c.bufr, fh, c.server.Codec)
			// it's ok to continue, when reached the EOF
			if err != nil && err != io.EOF {
				// TODO: log error instead?
				panic(err)
			}

			if req != nil {
				req.conn = c
				// handle the message to router
				w := NewResponseWriter(c.bufw, &ProtobufCodec{})

				if c.server.Handler != nil {
					c.server.Handler.ServeProto(w, req)
				}

				// flush bufw, if any
				// TODO: validation?
				if w.Buffered() > 0 {
					if d := c.server.WriteTimeout; d != 0 {
						c.rwc.SetWriteDeadline(time.Now().Add(d))
					}

					if err := w.Flush(); err != nil {
						panic(err)
					}
				}

				// if the writer require close, then return and close the conn
				if !w.KeepAlive() {
					return
				}
			}
		}

		// set rwc to idle state again
		c.setState(c.rwc, StateIdle)
		// handle connection idle
		if d := c.server.idleTimeout(); d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
			if _, err := c.bufr.Peek(4); err != nil {
				return
			}
		}
		c.rwc.SetReadDeadline(time.Time{})
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
	maxFrameSize    = 4096 - 1
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

// Help functions for reading and writing frame

// ReadFrameHeader from the io reader.
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

// ReadFrameBody from the io reader and frame header
// it will return a request if succeed
func ReadFrameBody(r io.Reader, fh FrameHeader, codec Codec) (req *Request, err error) {
	// read frame body
	// TODO: byte array pooling?
	fb := make([]byte, fh.Length)
	_, err = io.ReadFull(r, fb)

	if err != nil {
		return nil, err
	}

	req = &Request{}
	err = codec.Unmarshal(fb, req)

	if err != nil {
		return nil, err
	}

	return
}

// ... for test only
func ReadFrame(r io.Reader, codec Codec) (*Request, error) {
	fh, err := ReadFrameHeader(r)
	if err != nil {
		return nil, err
	}

	return ReadFrameBody(r, fh, codec)
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

	// frame body bytearray pooling
	// bytePool *BytePool = NewBytePool(516, maxFrameSize)
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

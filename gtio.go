package gotham

import (
	"errors"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var errClosing = errors.New("closing")
var errCloseConns = errors.New("close conns")

type gtServer struct {
	events   Events         // user events
	loops    []*gtLoop      // all the loops
	ln       net.Listener   // all the listeners
	loopwg   sync.WaitGroup // loop close waitgroup
	lnwg     sync.WaitGroup // listener close waitgroup
	cond     *sync.Cond     // shutdown signaler
	serr     error          // signal error
	accepted uintptr        // accept counter
}

type gtLoop struct {
	idx   int              // loop index
	ch    chan interface{} // command channel
	conns map[*gtConn]bool // track all the conns bound to this loop
}

type gtConn struct {
	conn   net.Conn    // original connection
	ctx    interface{} // user-defined context
	loop   *gtLoop     // owner loop
	donein []byte      // extra data for done connection
	done   int32       // 0: attached, 1: closed, 2: detached
}

type wakeReq struct {
	c *gtConn
}

func (c *gtConn) Context() interface{}       { return c.ctx }
func (c *gtConn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *gtConn) LocalAddr() net.Addr        { return c.conn.LocalAddr() }
func (c *gtConn) RemoteAddr() net.Addr       { return c.conn.RemoteAddr() }
func (c *gtConn) Wake()                      { c.loop.ch <- wakeReq{c} }

type gtIn struct {
	c  *gtConn
	in []byte
}

type gtErr struct {
	c   *gtConn
	err error
}

// waitForShutdown waits for a signal to shutdown
func (s *gtServer) waitForShutdown() error {
	s.cond.L.Lock()
	s.cond.Wait()
	err := s.serr
	s.cond.L.Unlock()
	return err
}

// signalShutdown signals a shutdown an begins server closing
func (s *gtServer) signalShutdown(err error) {
	s.cond.L.Lock()
	s.serr = err
	s.cond.Signal()
	s.cond.L.Unlock()
}

func gtServe(events Events, listener net.Listener) error {
	numLoops := events.NumLoops
	// if numLoops > 1, then set to NumCPU
	if numLoops <= 0 {
		if numLoops == 0 {
			numLoops = 1
		} else {
			numLoops = runtime.NumCPU()
		}
	}

	s := &gtServer{}
	s.events = events
	s.ln = listener

	// server shutdown signaler
	s.cond = sync.NewCond(&sync.Mutex{})

	// serving events handler called here
	if events.Serving != nil {
		var svr Server
		svr.NumLoops = numLoops
		svr.Addr = listener.Addr()
		action := events.Serving(svr)
		switch action {
		case Shutdown:
			return nil
		}
	}

	for i := 0; i < numLoops; i++ {
		s.loops = append(s.loops, &gtLoop{
			idx:   i,
			ch:    make(chan interface{}),
			conns: make(map[*gtConn]bool),
		})
	}
	var ferr error
	defer func() {
		// wait on a signal for shutdown
		ferr = s.waitForShutdown()

		// notify all loops to close by closing all listeners
		for _, l := range s.loops {
			l.ch <- errClosing
		}

		// wait on all loops to main loop channel events
		s.loopwg.Wait()

		// shutdown all listeners
		s.ln.Close()

		// wait on all listeners to complete
		s.lnwg.Wait()

		// close all connections
		s.loopwg.Add(len(s.loops))
		for _, l := range s.loops {
			l.ch <- errCloseConns
		}
		s.loopwg.Wait()

	}()

	// add waitgroup num
	s.loopwg.Add(numLoops)

	for i := 0; i < numLoops; i++ {
		go gtLoopRun(s, s.loops[i])
	}

	go gtListenerRun(s)

	return ferr
}

func gtListenerRun(s *gtServer) {
	var ferr error
	defer func() {
		s.signalShutdown(ferr)
		s.lnwg.Done()
	}()
	for {
		// tcp
		conn, err := s.ln.Accept()
		if err != nil {
			ferr = err
			return
		}
		// get the loop, load balance here
		l := s.loops[int(atomic.AddUintptr(&s.accepted, 1))%len(s.loops)]
		// TODO: sync.pool needed here
		c := &gtConn{conn: conn, loop: l}
		l.ch <- c
		go func(c *gtConn) {
			var packet [0xFFFF]byte
			for {
				n, err := c.conn.Read(packet[:])
				if err != nil {
					c.conn.SetReadDeadline(time.Now())
					l.ch <- &gtErr{c, err}
					return
				}
				l.ch <- &gtIn{c, append([]byte{}, packet[:n]...)}
			}
		}(c)
	}
}

func gtLoopRun(s *gtServer, l *gtLoop) {
	var err error
	defer func() {
		// loop stopped
		// why there is two wg.Done() here ?
		s.signalShutdown(err)
		s.loopwg.Done()
		gtLoopEgress(s, l)
		s.loopwg.Done()
	}()
	//fmt.Println("-- loop started --", l.idx)
	for {
		select {
		case v := <-l.ch:
			switch v := v.(type) {
			case error:
				err = v
			case *gtConn:
				err = gtLoopAccept(s, l, v)
			case *gtIn:
				err = gtLoopRead(s, l, v.c, v.in)
			case *gtErr:
				err = gtLoopError(s, l, v.c, v.err)
			case wakeReq:
				err = gtLoopRead(s, l, v.c, nil)
			}
		}
		if err != nil {
			return
		}
	}
}

func gtLoopEgress(s *gtServer, l *gtLoop) {
	var closed bool
loop:
	for v := range l.ch {
		switch v := v.(type) {
		case error:
			if v == errCloseConns {
				closed = true
				for c := range l.conns {
					gtLoopClose(s, l, c)
				}
			}
		case *gtErr:
			gtLoopError(s, l, v.c, v.err)
		}
		if len(l.conns) == 0 && closed {
			break loop
		}
	}
}

func gtLoopError(s *gtServer, l *gtLoop, c *gtConn, err error) error {
	delete(l.conns, c)
	closeEvent := true
	switch atomic.LoadInt32(&c.done) {
	case 0: // read error
		c.conn.Close()
		if err == io.EOF {
			err = nil
		}
	case 1: // closed
		c.conn.Close()
		err = nil
	case 2: // detached
		err = nil
		if s.events.Detached == nil {
			c.conn.Close()
		} else {
			closeEvent = false
			switch s.events.Detached(c, &gtDetachedConn{c.conn, c.donein}) {
			case Shutdown:
				return errClosing
			}
		}
	}
	if closeEvent {
		if s.events.Closed != nil {
			switch s.events.Closed(c, err) {
			case Shutdown:
				return errClosing
			}
		}
	}
	return nil
}

type gtDetachedConn struct {
	conn net.Conn // original conn
	in   []byte   // extra input data
}

func (c *gtDetachedConn) Read(p []byte) (n int, err error) {
	if len(c.in) > 0 {
		if len(c.in) <= len(p) {
			copy(p, c.in)
			n = len(c.in)
			c.in = nil
			return
		}
		copy(p, c.in[:len(p)])
		n = len(p)
		c.in = c.in[n:]
		return
	}
	return c.conn.Read(p)
}

func (c *gtDetachedConn) Write(p []byte) (n int, err error) {
	return c.conn.Write(p)
}

func (c *gtDetachedConn) Close() error {
	return c.conn.Close()
}

func (c *gtDetachedConn) Wake() {}

func gtLoopRead(s *gtServer, l *gtLoop, c *gtConn, in []byte) error {
	if atomic.LoadInt32(&c.done) == 2 {
		// should not ignore reads for detached connections
		c.donein = append(c.donein, in...)
		return nil
	}
	if s.events.Data != nil {
		out, action := s.events.Data(c, in)
		if len(out) > 0 {
			if s.events.PreWrite != nil {
				s.events.PreWrite()
			}
			c.conn.Write(out)
		}
		switch action {
		case Shutdown:
			return errClosing
		case Detach:
			return gtLoopDetach(s, l, c)
		case Close:
			return gtLoopClose(s, l, c)
		}
	}
	return nil
}

func gtLoopDetach(s *gtServer, l *gtLoop, c *gtConn) error {
	atomic.StoreInt32(&c.done, 2)
	c.conn.SetReadDeadline(time.Now())
	return nil
}

func gtLoopClose(s *gtServer, l *gtLoop, c *gtConn) error {
	atomic.StoreInt32(&c.done, 1)
	c.conn.SetReadDeadline(time.Now())
	return nil
}

func gtLoopAccept(s *gtServer, l *gtLoop, c *gtConn) error {
	l.conns[c] = true

	if s.events.Opened != nil {
		out, opts, action := s.events.Opened(c)
		if len(out) > 0 {
			if s.events.PreWrite != nil {
				s.events.PreWrite()
			}
			c.conn.Write(out)
		}
		if opts.TCPKeepAlive > 0 {
			if c, ok := c.conn.(*net.TCPConn); ok {
				c.SetKeepAlive(true)
				c.SetKeepAlivePeriod(opts.TCPKeepAlive)
			}
		}
		switch action {
		case Shutdown:
			return errClosing
		case Detach:
			return gtLoopDetach(s, l, c)
		case Close:
			return gtLoopClose(s, l, c)
		}
	}
	return nil
}

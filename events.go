package gotham

import (
	"io"
	"sync/atomic"
	"time"

	// "github.com/luci/luci-go/common/data/recordio"
	"github.com/sirupsen/logrus"
	"github.com/sleep2death/evio"
)

var accepted int32

// Count all the connections number
func Count() int32 {
	return atomic.LoadInt32(&accepted)
}

func initEvents(numLoops int) evio.Events {
	var events evio.Events
	events.NumLoops = numLoops

	events.Serving = serving

	events.Opened = opened
	events.Closed = closed

	events.Detached = detached

	events.Data = data
	events.Tick = tick

	return events
}

// Serving fires when the server can accept connections. The server
// parameter has information and various utilities.
func serving(server evio.Server) (action evio.Action) {
	log.WithFields(logrus.Fields{
		"address":  server.Addrs,
		"numloops": server.NumLoops,
	}).Debug("Start serving")
	return
}

// Opened fires when a new connection has opened.
// The info parameter has information about the connection such as
// it's local and remote address.
// Use the out return value to write data to the connection.
// The opts return value is used to set connection options.
func opened(c evio.Conn) (out []byte, opts evio.Options, action evio.Action) {
	log.WithFields(logrus.Fields{
		"local":     c.LocalAddr(),
		"remote":    c.RemoteAddr(),
		"addrIndex": c.AddrIndex(),
	}).Debug("Accepet a client")

	opts.TCPKeepAlive = time.Minute * 5
	opts.ReuseInputBuffer = true

	c.SetContext(&evio.InputStream{})

	atomic.AddInt32(&accepted, 1)

	return
}

// Closed fires when a connection has closed.
// The err parameter is the last known connection error.
func closed(c evio.Conn, err error) (action evio.Action) {
	log.Debug("connection closed")

	if err != nil {
		log.Errorln("connection error:", err)
	}

	atomic.AddInt32(&accepted, -1)
	return
}

// Detached fires when a connection has been previously detached.
// Once detached it's up to the receiver of this event to manage the
// state of the connection. The Closed event will not be called for
// this connection.
// The conn parameter is a ReadWriteCloser that represents the
// underlying socket connection. It can be freely used in goroutines
// and should be closed when it's no longer needed.
func detached(c evio.Conn, rwc io.ReadWriteCloser) (action evio.Action) {
	return
}

// Data fires when a connection sends the server data.
// The in parameter is the incoming data.
// Use the out return value to write data to the connection.
func data(c evio.Conn, in []byte) (out []byte, action evio.Action) {
	if in == nil {
		return
	}

	is := c.Context().(*evio.InputStream)
	data := is.Begin(in)
	msgs, leftover := readFrames(data)

	for _, msg := range msgs {
		log.Info(string(msg))
	}

	if len(leftover) > 0 {
		is.End(leftover)
	}

	// log.Info(string(f))

	return
}

// Tick fires immediately after the server starts and will fire again
// following the duration specified by the delay return value.
func tick() (delay time.Duration, action evio.Action) {
	return
}

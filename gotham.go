package gotham

import (
	"fmt"
	"io"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/evio"
)

var log = logrus.New()
var numConn int32

// Config of the server
type Config struct {
	Network   string // tcp tcp4 tcp6 udp udp4 udp6 unix, default is tcp
	Address   string // the address of the server, default is localhost
	Port      int    // the port of the server, default is 8101
	ReusePort bool   // the SO_REUSEPORT socket option, not available on windows: [details] https://stackoverflow.com/questions/14388706/socket-options-so-reuseaddr-and-so-reuseport-how-do-they-differ-do-they-mean-t
	Stdlib    bool   // use the standard net package or not, default is false(using syscall to call system network functions

	NumLoops    int              // multithread if the value is larger than 1, default is based on your cpu cores
	LoadBalance evio.LoadBalance // loadbalance strategy, not work if the NumLoops is less than 1
}

func (config *Config) getAddr() (addr string) {
	scheme := config.Network
	if config.Stdlib {
		scheme += "-net"
	}
	addr = fmt.Sprintf("%s://%s:%d?reusePort=%t", scheme, config.Address, config.Port, config.ReusePort)
	return
}

// Default get the default server
func Default() *Config {
	log.Level = logrus.DebugLevel

	config := &Config{
		Network:   "tcp",
		Address:   "localhost",
		Port:      8101,
		ReusePort: true,
		Stdlib:    false,

		NumLoops:    runtime.NumCPU(),
		LoadBalance: evio.Random,
	}

	return config
}

// Serve start the server
func Serve(config *Config) (err error) {

	events := initEvents(config.NumLoops)
	address := config.getAddr()

	err = evio.Serve(events, address)
	return
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

// Count the connected conns
func Count() int32 {
	return atomic.LoadInt32(&numConn)
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

	atomic.AddInt32(&numConn, 1)

	return
}

func closed(c evio.Conn, err error) (action evio.Action) {
	atomic.AddInt32(&numConn, -1)

	log.Debug("connection closed")

	if err != nil {
		log.Errorln("connection error:", err)
	}

	return
}

func detached(c evio.Conn, rwc io.ReadWriteCloser) (action evio.Action) {
	return
}

func data(c evio.Conn, in []byte) (out []byte, action evio.Action) {
	log.Info(string(in))
	return
}

func tick() (delay time.Duration, action evio.Action) {
	return
}

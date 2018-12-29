package gotham

import (
	"io"
	"net"
	"time"
)

// Action is an action that occurs after the completion of an event.
type Action int

const (
	// None indicates that no action should occur following an event.
	None Action = iota
	// Detach detaches a connection. Not available for UDP connections.
	Detach
	// Close closes the connection.
	Close
	// Shutdown shutdowns the server.
	Shutdown
)

// Options are set when the client opens.
type Options struct {
	// TCPKeepAlive (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration
	// ReuseInputBuffer will forces the connection to share and reuse the
	// same input packet buffer with all other connections that also use
	// this option.
	// Default value is false, which means that all input data which is
	// passed to the Data event will be a uniquely copied []byte slice.
	ReuseInputBuffer bool
}

// Server represents a server context which provides information about the
// running server and has control functions for managing state.
type Server struct {
	// with the addr object passed to the Serve function.
	Addr net.Addr
	// NumLoops is the number of loops that the server is using.
	NumLoops int
}

// Conn is an evio connection.
type Conn interface {
	// Context returns a user-defined context.
	Context() interface{}
	// SetContext sets a user-defined context.
	SetContext(interface{})
	// LocalAddr is the connection's local socket address.
	LocalAddr() net.Addr
	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() net.Addr
	// Wake triggers a Data event for this connection.
	Wake()
}

// LoadBalance sets the load balancing method.
type LoadBalance int

const (
	// Random requests that connections are randomly distributed.
	Random LoadBalance = iota
	// RoundRobin requests that connections are distributed to a loop in a
	// round-robin fashion.
	RoundRobin
	// LeastConnections assigns the next accepted connection to the loop with
	// the least number of active connections.
	LeastConnections
)

// Events represents the server events for the Serve call.
// Each event has an Action return value that is used manage the state
// of the connection and server.
type Events struct {
	// NumLoops sets the number of loops to use for the server. Setting this
	// to a value greater than 1 will effectively make the server
	// multithreaded for multi-core machines. Which means you must take care
	// with synchonizing memory between all event callbacks. Setting to 0 or 1
	// will run the server single-threaded. Setting to -1 will automatically
	// assign this value equal to runtime.NumProcs().
	NumLoops int
	// LoadBalance sets the load balancing method. Load balancing is always a
	// best effort to attempt to distribute the incoming connections between
	// multiple loops. This option is only works when NumLoops is set.
	LoadBalance LoadBalance
	// Serving fires when the server can accept connections. The server
	// parameter has information and various utilities.
	Serving func(server Server) (action Action)
	// Opened fires when a new connection has opened.
	// The info parameter has information about the connection such as
	// it's local and remote address.
	// Use the out return value to write data to the connection.
	// The opts return value is used to set connection options.
	Opened func(c Conn) (out []byte, opts Options, action Action)
	// Closed fires when a connection has closed.
	// The err parameter is the last known connection error.
	Closed func(c Conn, err error) (action Action)
	// Detached fires when a connection has been previously detached.
	// Once detached it's up to the receiver of this event to manage the
	// state of the connection. The Closed event will not be called for
	// this connection.
	// The conn parameter is a ReadWriteCloser that represents the
	// underlying socket connection. It can be freely used in goroutines
	// and should be closed when it's no longer needed.
	Detached func(c Conn, rwc io.ReadWriteCloser) (action Action)
	// PreWrite fires just before any data is written to any client socket.
	PreWrite func()
	// Data fires when a connection sends the server data.
	// The in parameter is the incoming data.
	// Use the out return value to write data to the connection.
	Data func(c Conn, in []byte) (out []byte, action Action)
	// Tick fires immediately after the server starts and will fire again
	// following the duration specified by the delay return value.
	Tick func() (delay time.Duration, action Action)
}

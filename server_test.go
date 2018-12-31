package gotham

import (
	"math/rand"
	"net"
	"testing"
	"time"
)

func TestServe(t *testing.T) {
	addr := ":4000"
	ln, err := net.Listen("tcp", addr)

	if err != nil {
		panic(err)
	}

	// connection count
	// var count uint32

	// evts := Events{
	// 	Serving: func(server Server) (action Action) {
	// 		assert.Equal(t, server.Addr.Network(), "tcp")
	// 		assert.Equal(t, server.Addr.String(), addr)
	// 		return None
	// 	},
	// 	Opened: func(c Conn) (out []byte, opts Options, action Action) {
	// 		atomic.AddUint32(&count, 1)
	// 		opts = Options{
	// 			ReuseInputBuffer: true,
	// 			TCPKeepAlive:     time.Minute * 5,
	// 		}
	// 		action = None
	// 		out = nil
	// 		return
	// 	},
	// 	Data: func(c Conn, in []byte) (out []byte, action Action) {
	// 		t.Log(string(in))
	// 		return
	// 	},
	// }

	server := &Server{}
	go server.Serve(ln)
	t.Log("serving start at >>>", addr)

	numClients := 100

	for i := 0; i < numClients; i++ {
		conn, _ := net.DialTimeout("tcp", addr, time.Minute*5)
		if r := rand.Intn(2); r == 1 {
			conn.Write([]byte("Hello"))
		} else {
			conn.Write([]byte("GoodBye"))
		}
	}

	time.Sleep(time.Second * 1)
	// assert.Equal(t, uint32(numClients), atomic.LoadUint32(&count))
}

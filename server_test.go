package gotham

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	addr1                       = ":4000"
	addr2                       = ":4001"
	countA, countB, totalWrites int32
	dialCount                   = 50 // clients num
	writeCount                  = 50 // write num with each client
	stopChan                    = make(chan struct{})
	dialInterval                = time.Millisecond * 1
	writeInterval               = time.Millisecond * 1
	testDuration                = time.Millisecond * 75
)

func TestServeRead(t *testing.T) {
	ln1, err := net.Listen("tcp", addr1)
	ln2, err := net.Listen("tcp", addr2)

	if err != nil {
		panic(err)
	}

	server := &Server{}
	server.ReadTimeout = time.Minute

	server.ServeTCP = func(w io.Writer, fh FrameHeader, fb []byte) {
		str := string(fb)

		if strings.Index(str, "Hello") >= 0 {
			atomic.AddInt32(&countA, -1)
		} else if strings.Index(str, "Goodbye") >= 0 {
			atomic.AddInt32(&countB, -1)
		} else {
			t.Log("invalid data")
		}
	}

	listen := func(ln net.Listener) {
		if err = server.Serve(ln); err != nil {
			// it is going to throw an error, when the server finally closed
			fmt.Println(ln.Addr(), err.Error())
		}
	}

	// serve two listners
	go listen(ln1)
	go listen(ln2)

	// interval := time.Millisecond // write and connect interval
	// ticker := time.NewTicker(interval)

	go dial(LoopWrite)

	// not enough time to complete the data writing,
	// so we can test the shutdown func is going to work properly
	time.Sleep(testDuration)
	// stop all writing...
	close(stopChan)
	t.Logf("total message writes: %d/%d", atomic.LoadInt32(&totalWrites), dialCount*writeCount)
	// shutdown all clients goroutines
	_ = server.Shutdown()

	// plus one when writer a msg, minus one when read,
	// so if all the write/read(s) are functional, the count should be ZERO
	assert.Equal(t, int32(0), atomic.LoadInt32(&countA))
	assert.Equal(t, int32(0), atomic.LoadInt32(&countB))
}

type dialType uint

const (
	LoopWrite dialType = iota
	Echo
)

func dial(t dialType) {
	for i := 0; i < dialCount; i++ {
		var conn net.Conn
		var err error

		if l := rand.Intn(2); l == 1 {
			conn, err = net.DialTimeout("tcp", addr1, time.Minute*5)
		} else {
			conn, err = net.DialTimeout("tcp", addr2, time.Minute*5)
		}
		// if connection refused, then stop
		if err != nil {
			return
		}

		w := bufio.NewWriter(conn)
		switch t {
		case LoopWrite:
			go writeLoop(w)
		case Echo:
			_ = WriteData(w, []byte("PING"))
			_ = w.Flush()
			go read(w, bufio.NewReader(conn))
		default:
		}

		time.Sleep(dialInterval)
	}
}

func writeLoop(w *bufio.Writer) {
	for j := 0; j < writeCount; j++ {
		select {
		case <-stopChan:
			// fmt.Println("stop chan revieced")
			return
		default:
		}

		var data []byte
		if r := rand.Intn(2); r == 1 {
			data = []byte("Hello")
			atomic.AddInt32(&countA, 1)
		} else {
			data = []byte("Goodbye")
			atomic.AddInt32(&countB, 1)
		}

		_ = WriteData(w, data)
		_ = w.Flush()

		atomic.AddInt32(&totalWrites, 1)
		time.Sleep(writeInterval)
	}
}

func TestEcho(t *testing.T) {
	stopChan = make(chan struct{})
	atomic.StoreInt32(&countA, 0)
	atomic.StoreInt32(&countB, 0)

	ln1, err := net.Listen("tcp", addr1)
	ln2, err := net.Listen("tcp", addr2)

	if err != nil {
		panic(err)
	}

	server := &Server{}
	server.ReadTimeout = time.Minute

	server.ServeTCP = func(w io.Writer, fh FrameHeader, fb []byte) {
		if str := string(fb); str == "PING" {
			time.Sleep(writeInterval)
			WriteData(w, []byte("PONG"))
			w.(*bufio.Writer).Flush()
			atomic.AddInt32(&countB, 1)
		}
	}

	listen := func(ln net.Listener) {
		if err = server.Serve(ln); err != nil {
			// it is going to throw an error, when the server finally closed
			fmt.Println(ln.Addr(), err.Error())
		}
	}

	// serve two listners
	go listen(ln1)
	go listen(ln2)

	go dial(Echo)

	time.Sleep(time.Second * 2)

	close(stopChan)
	server.Shutdown()
	t.Logf("PING count:%d, PONG count:%d", atomic.LoadInt32(&countA), atomic.LoadInt32(&countB))
}

func createServer() *Server {
	server := &Server{}
	server.ReadTimeout = time.Minute

	return server
}

func listen(server *Server, ln net.Listener) {
	if err := server.Serve(ln); err != nil {
		// it is going to throw an error, when the server finally closed
		fmt.Println(ln.Addr(), err.Error())
	}
}

func read(w *bufio.Writer, r *bufio.Reader) {
	for {
		select {
		case <-stopChan:
			// fmt.Println("stop chan revieced")
			return
		default:
		}
		fh, err := ReadFrameHeader(r)

		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			panic(err)
		}

		fb := make([]byte, fh.Length)

		_, err = io.ReadFull(r, fb)

		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			panic(err)
		}

		if str := string(fb); str == "PONG" {
			time.Sleep(writeInterval * 15)

			if err = WriteData(w, []byte("PING")); err != nil {
				panic(err)
			}

			w.Flush()

			atomic.AddInt32(&countA, 1)
		}
	}
}

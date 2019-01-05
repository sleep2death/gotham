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

func TestServe(t *testing.T) {
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

	go dial()

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

func dial() {
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
		go write(w)
		time.Sleep(dialInterval)
	}
}

func write(w *bufio.Writer) {
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

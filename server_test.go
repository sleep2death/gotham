package gotham

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	var countA int32
	var countB int32

	addr1 := ":4000"
	ln1, err := net.Listen("tcp", addr1)

	addr2 := ":4001"
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
		}
	}

	listen := func(ln net.Listener) {
		if err = server.Serve(ln); err != nil {
			// it is going to throw an error, when the server finally closed
			fmt.Println(ln.Addr(), err.Error())
		}
	}

	go listen(ln1)
	go listen(ln2)

	numClients := 2500                 // clients num
	numWrites := 100                   // write num with each client
	interval := time.Millisecond * 100 // write interval

	go func() {
		for i := 0; i < numClients; i++ {
			var conn net.Conn

			if l := rand.Intn(2); l == 1 {
				conn, err = net.DialTimeout("tcp", addr1, time.Minute*5)
			} else {
				conn, err = net.DialTimeout("tcp", addr2, time.Minute*5)
			}

			if err != nil {
				panic(err)
			}

			for j := 0; j < numWrites; j++ {
				r := rand.Intn(2)
				var data []byte
				writer := bufio.NewWriter(conn)
				go func(i int, j int) {
					if r = rand.Intn(2); r == 1 {
						data = []byte("Hello>" + strconv.Itoa(i) + "-" + strconv.Itoa(j))
						atomic.AddInt32(&countA, 1)
					} else {
						data = []byte("Goodbye>" + strconv.Itoa(i) + "-" + strconv.Itoa(j))
						atomic.AddInt32(&countB, 1)
					}

					_ = WriteData(writer, data)
					time.Sleep(interval)
				}(i, j)
				time.Sleep(interval)
			}
		}
	}()

	// not enough time to complete the data writing,
	// so we can test the shutdown func is going to work properly
	time.Sleep(time.Millisecond * 1200)

	_ = server.Shutdown()

	assert.Equal(t, len(server.activeConn), 0)

	// plus one when writer a msg, minus one when read,
	// so if all the write/read(s) are functional, the count should be ZERO
	assert.Equal(t, atomic.LoadInt32(&countA), int32(0))
	assert.Equal(t, atomic.LoadInt32(&countB), int32(0))
}

func WriteFrame(msg []byte) (data []byte) {
	sizeBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(sizeBuf, uint16(len(msg)))

	msg = append(sizeBuf, msg...)
	data = append(data, msg...)
	return
}

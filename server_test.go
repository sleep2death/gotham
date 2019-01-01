package gotham

import (
	"encoding/binary"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServe(t *testing.T) {
	addr1 := ":4000"
	ln1, err := net.Listen("tcp", addr1)

	addr2 := ":4002"
	ln2, err := net.Listen("tcp", addr2)

	if err != nil {
		panic(err)
	}

	server := &Server{}
	server.ReadTimeout = time.Minute

	go server.Serve(ln1)
	go server.Serve(ln2)

	numClients := 2
	numWrites := 100

	go func() {
		for i := 0; i < numClients; i++ {
			var conn net.Conn

			if l := rand.Intn(2); l == 1 {
				conn, _ = net.DialTimeout("tcp", addr1, time.Minute*5)
			} else {
				conn, _ = net.DialTimeout("tcp", addr2, time.Minute*5)
			}

			go func(i int) {
				for j := 0; j < numWrites; j++ {
					if r := rand.Intn(2); r == 1 {
						data := []byte("Hello > " + strconv.Itoa(i) + "-" + strconv.Itoa(j))
						// conn.Write(WriteFrame(data))
						WriteData(conn, data)
					} else {
						data := []byte("Goodbye > " + strconv.Itoa(i) + "-" + strconv.Itoa(j))
						// conn.Write(WriteFrame(data))
						WriteData(conn, data)
					}

					time.Sleep(time.Millisecond * 10)
				}
			}(i)
		}
	}()

	time.Sleep(time.Millisecond * 200)

	server.Shutdown()

	assert.Equal(t, len(server.activeConn), 0)
}

func WriteFrame(msg []byte) (data []byte) {
	sizeBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(sizeBuf, uint16(len(msg)))

	msg = append(sizeBuf, msg...)
	data = append(data, msg...)
	return
}

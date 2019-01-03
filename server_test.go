package gotham

import (
	"bufio"
	"encoding/binary"
	"io"
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

	server.ServeTCP = func(w io.Writer, fh FrameHeader, fb []byte) {
	}

	go server.Serve(ln1)
	go server.Serve(ln2)

	numClients := 3
	numWrites := 10
	interval := time.Millisecond * 10

	go func() {
		for i := 0; i < numClients; i++ {
			var conn net.Conn

			if l := rand.Intn(2); l == 1 {
				conn, _ = net.DialTimeout("tcp", addr1, time.Minute*5)
			} else {
				conn, _ = net.DialTimeout("tcp", addr2, time.Minute*5)
			}

			for j := 0; j < numWrites; j++ {
				r := rand.Intn(2)
				var data []byte
				writer := bufio.NewWriter(conn)
				go func(i int, j int) {
					if r = rand.Intn(2); r == 1 {
						data = []byte("Hello>" + strconv.Itoa(i) + "-" + strconv.Itoa(j))
					} else {
						data = []byte("Goodbye>" + strconv.Itoa(i) + "-" + strconv.Itoa(j))
					}

					WriteData(writer, data)
					writer.Flush()

					time.Sleep(interval)
				}(i, j)
			}
		}
	}()

	time.Sleep(time.Millisecond * 2000)

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

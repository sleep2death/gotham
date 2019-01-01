package main

import (
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/sleep2death/gotham"
)

func main() {
	addr1 := ":4000"
	ln1, err := net.Listen("tcp", addr1)

	addr2 := ":4002"
	ln2, err := net.Listen("tcp", addr2)

	if err != nil {
		panic(err)
	}

	server := &gotham.Server{}
	server.ReadTimeout = time.Minute

	go server.Serve(ln1)
	go server.Serve(ln2)

	numClients := 2
	numWrites := 3

	go func() {
		for i := 0; i < numClients; i++ {
			var conn net.Conn

			if l := rand.Intn(2); l == 1 {
				conn, _ = net.DialTimeout("tcp", addr1, time.Minute*5)
			} else {
				conn, _ = net.DialTimeout("tcp", addr2, time.Minute*5)
			}

			for j := 0; j < numWrites; j++ {
				if r := rand.Intn(2); r == 1 {
					data := []byte("Hello > " + strconv.Itoa(i) + "-" + strconv.Itoa(j))
					conn.Write(gotham.WriteFrame(data))
				} else {
					data := []byte("Goodbye > " + strconv.Itoa(i) + "-" + strconv.Itoa(j))
					conn.Write(gotham.WriteFrame(data))
				}
			}
		}
	}()

	server.Shutdown()
}

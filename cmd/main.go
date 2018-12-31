package main

import (
	"math/rand"
	"net"
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
	server.IdleTimeout = time.Minute * 5

	go server.Serve(ln1)
	go server.Serve(ln2)

	numClients := 10

	for i := 0; i < numClients; i++ {
		var conn net.Conn

		if l := rand.Intn(2); l == 1 {
			conn, _ = net.DialTimeout("tcp", addr1, time.Minute*5)
		} else {
			conn, _ = net.DialTimeout("tcp", addr2, time.Minute*5)
		}

		for j := 0; j < 1; j++ {
			if r := rand.Intn(2); r == 1 {
				data := []byte("Hello")
				conn.Write(gotham.WriteFrame(data))
			} else {
				data := []byte("GoodBye")
				conn.Write(gotham.WriteFrame(data))
			}
		}
	}

	time.Sleep(time.Second * 5)

	// go server.Shutdown()
}

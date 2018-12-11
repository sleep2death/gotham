package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/sleep2death/gotham"
)

func main() {
	cfg := gotham.Default()
	go func() {
		err := gotham.Serve(cfg)
		if err != nil {
			log.Panic(err)
		}
	}()

	var conns []net.Conn

	for count := 0; count < 10; count++ {
		conn, err := net.Dial("tcp", "localhost:8101")
		conns = append(conns, conn)

		if err != nil {
			log.Panic(err)
		}

		// time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	}

	for index, conn := range conns {
		str := fmt.Sprintf("Hello, Gotham, from conn: %d\n", index)
		conn.Write([]byte(str))
	}

	time.Sleep(time.Second * 10)

	fmt.Println(gotham.Count())
}

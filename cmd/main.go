package main

import (
	"log"
	"math/rand"
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

	count := 0
	for count < 10 {
		_, err := net.Dial("tcp", "localhost:8101")

		if err != nil {
			log.Panic(err)
		}

		time.Sleep(time.Second * time.Duration(rand.Intn(3)))

		count++
	}

	time.Sleep(time.Second)
}

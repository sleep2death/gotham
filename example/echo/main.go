package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/sleep2death/gotham"
)

func main() {
	addr := ":4000"

	ln, err := net.Listen("tcp", addr)

	if err != nil {
		panic(err)
	}

	server := &gotham.Server{}

	server.ServeTCP = func(w io.Writer, fh gotham.FrameHeader, fb []byte) {
		if str := string(fb); str == "PING" {
			// passive write back, when received from clients
			_ = gotham.WriteData(w, []byte("PONG"))
			_ = w.(*bufio.Writer).Flush()
			fmt.Println("Pong")
		}
	}

	// start the server
	go func() {
		if err = server.Serve(ln); err != nil {
			fmt.Println(err)
		}
	}()

	// create client
	conn, err := net.DialTimeout("tcp", addr, time.Minute*5)

	// if connection refused, then stop
	if err != nil {
		return
	}

	w := bufio.NewWriter(conn)
	_ = gotham.WriteData(w, []byte("PING"))
	_ = w.Flush()

	stopChan := make(chan struct{})
	go read(w, bufio.NewReader(conn), stopChan)

	time.Sleep(time.Second * 10)

	close(stopChan)
	_ = server.Shutdown()
}

func read(w *bufio.Writer, r *bufio.Reader, stopChan chan struct{}) {
	for {
		fh, err := gotham.ReadFrameHeader(r)

		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			fmt.Println(err)
			return
		}

		fb := make([]byte, fh.Length)

		_, err = io.ReadFull(r, fb)

		// it's ok to continue, when reached the EOF
		if err != nil && err != io.EOF {
			fmt.Println(err)
			return
		}

		if str := string(fb); str == "PONG" {
			// stop ping back, when stop channel closed
			select {
			case <-stopChan:
				fmt.Println("stop writing")
				return
			default:
			}

			if err = gotham.WriteData(w, []byte("PING")); err != nil {
				fmt.Println(err)
				return
			}

			_ = w.Flush()
			fmt.Println("Ping")
			time.Sleep(time.Second)
		}
	}
}

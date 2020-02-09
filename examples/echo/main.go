package main

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"github.com/sleep2death/gotham"
	"github.com/sleep2death/gotham/examples/pb"
)

func main() {
	// SERVER
	// Starts a new gotham instance without any middleware.
	router := gotham.New()

	// Define your handlers
	router.Handle("pb.EchoMessage", func(c *gotham.Context) {
		message := new(pb.EchoMessage)

		// If some error fires, you can abort the request.
		if err := proto.Unmarshal(c.Data(), message); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		// log.Printf("Ping request received at %s", ptypes.TimestampString(message.Ts))

		message.Message = "Pong"
		message.Ts = ptypes.TimestampNow()
		c.Write(message)
	})

	// Run, gotham, Run...
	addr := ":9090"
	go func() { log.Fatal(router.Run(addr)) }()

	// Wait a little while for server prepairing
	time.Sleep(time.Millisecond * 5)

	// CLIENTS
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Connect to server.
			client, err := net.Dial("tcp", addr)
			if err != nil {
				log.Fatal("can not get in touch with gotham.")
			}
			defer client.Close()

			msg := &pb.EchoMessage{
				Message: "Ping",
				Ts:      ptypes.TimestampNow(),
			}

			// Write the message with a little help of gotham utils function.
			if err := gotham.WriteFrame(client, msg); err != nil {
				log.Fatalf("client write data error: %s.", err)
			}

			res, err := gotham.ReadFrame(client)
			if err != nil {
				log.Fatalf("client read data error: %s.", err)
			}
			// Unmarshal the raw data
			err = proto.Unmarshal(res.Data(), msg)
			if err != nil {
				log.Fatalf("client read data error: %s.", err)
			}

			if msg.GetMessage() == "Pong" {
				log.Printf("Ping response of (%d) received.", idx)
			}
		}(i)
	}
	wg.Wait()
}

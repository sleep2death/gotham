package gotham

import (
	"log"
	"net"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestRouterServe(t *testing.T) {
	str := "Ping"

	r := New()
	group := r.Group("/gotham")
	group.Use(func(ctx *Context) {
		// log.Printf("[middleware]")
		assert.Equal(t, "/gotham/Ping", ctx.FullPath())
	})

	group.Handle("/Ping", func(ctx *Context) {
		// log.Printf("[%s]", ctx.FullPath())
		var msg Ping
		err := proto.Unmarshal(ctx.request.data, &msg)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, str, msg.GetMessage())
		msg.Message = "Pong"
		ctx.WriteAny("/gotham/Ping", &msg)
		// log.Printf("ping message: %s", msg.GetMessage())
	})

	go r.Run(":9001")

	time.Sleep(time.Millisecond)

	conn, err := net.Dial("tcp", ":9001")
	if err != nil {
		t.Fatal(err)
	}

	// write request
	log.Println("write data to server")

	w := newBufioWriter(conn)
	br := newBufioReader(conn)

	// write message frame to server
	pb := &Ping{Message: str}
	WriteFrame(w, pb)
	w.Flush()

	time.Sleep(time.Millisecond * 5)

	// read response
	fh, err := ReadFrameHeader(br)
	if err != nil {
		t.Fatal(err)
	}
	req, err := ReadFrameBody(br, fh)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "/gotham/Ping", req.url)

	resp := &Ping{}
	err = proto.Unmarshal(req.data, resp)
	assert.Equal(t, "Pong", resp.GetMessage())

	// reader := bufio.NewReader(conn)
	// packet, err = reader.Read()
	time.Sleep(time.Millisecond * 5)
}

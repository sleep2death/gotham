package gotham

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
)

func TestRouterServe(t *testing.T) {
	str := "Ping"

	r := New()
	group := r.Group("/gotham")
	group.Use(func(ctx *Context) {
		// log.Printf("[middleware]")
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
	pb := &Ping{Message: str}
	b, _ := proto.Marshal(pb)

	msg := &types.Any{
		TypeUrl: "/gotham/Ping",
		Value:   b,
	}

	packet, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	w := bufio.NewWriter(conn)
	br := bufio.NewReader(conn)

	err = WriteFrame(w, packet)
	if err != nil {
		t.Fatal(err)
	}
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

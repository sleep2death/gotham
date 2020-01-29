package gotham

import (
	"bufio"
	"log"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

func TestRouterServe(t *testing.T) {
	r := New()
	group := r.Group("/gotham")
	group.Use(func(ctx *Context) {
		log.Printf("[middleware]")
	})

	group.Handle("/PingMsg", func(ctx *Context) {
		log.Printf("[%s]", ctx.FullPath())
		var msg PingMsg
		err := proto.Unmarshal(ctx.request.data, &msg)
		if err != nil {
			panic(err)
		}
		log.Printf("ping message: %s", msg.GetMessage())
	})

	go r.Run(":9001")

	time.Sleep(time.Millisecond)

	conn, err := net.Dial("tcp", ":9001")
	if err != nil {
		t.Fatal(err)
	}

	pb := &PingMsg{Message: "Ping"}
	b, _ := proto.Marshal(pb)

	msg := &any.Any{
		TypeUrl: "/gotham/PingMsg",
		Value:   b,
	}

	packet, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	w := bufio.NewWriter(conn)
	err = WriteData(w, packet)

	if err != nil {
		t.Fatal(err)
	}

	w.Flush()
	time.Sleep(time.Millisecond * 5)

	// r.ServeProto(nil, &Request{URL: msg.TypeUrl})
}

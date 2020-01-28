package gotham

import (
	"log"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

func TestRouterServe(t *testing.T) {
	r := New()
	r.Handle("/gotham/PingMsg", func(ctx *Context) {
		log.Printf("[req: %s]", ctx.FullPath())
	})

	pb := &PingMsg{Message: "Ping"}
	b, _ := proto.Marshal(pb)

	msg := &any.Any{
		TypeUrl: "/gotham/PingMsg",
		Value:   b,
	}

	r.ServeProto(nil, msg)
}

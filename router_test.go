package gotham

import (
	"log"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

func TestRouterServe(t *testing.T) {
	r := New()
	group := r.Group("/gotham")

	group.Handle("/PingMsg", func(ctx *Context) {
		log.Printf("[req: %s]", ctx.FullPath())
	})

	pb := &PingMsg{Message: "Ping"}
	b, _ := proto.Marshal(pb)

	msg := &any.Any{
		TypeUrl: "/gotham/PingMsg",
		Value:   b,
	}

	r.Serve(nil, msg)
}

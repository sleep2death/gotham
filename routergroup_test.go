package gotham

import (
	fmt "fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestRouterGroupBasic(t *testing.T) {
	router := New()
	group := router.Group("/hola", func(c *Context) {})
	group.Use(func(c *Context) {})

	assert.Len(t, group.Handlers, 2)
	assert.Equal(t, "/hola", group.BasePath())
	assert.Equal(t, router, group.engine)

	group2 := group.Group("manu")
	group2.Use(func(c *Context) {}, func(c *Context) {})

	assert.Len(t, group2.Handlers, 4)
	assert.Equal(t, "/hola/manu", group2.BasePath())
	assert.Equal(t, router, group2.engine)
}

type recorder struct {
	Message proto.Message
	close   bool
}

func (rr *recorder) WriteMessage(msg proto.Message) error {
	rr.Message = msg
	return nil
}

func (rr *recorder) SetClose(value bool) {
	rr.close = value
}

func (rr *recorder) Close() bool {
	return rr.close
}

func TestRouterGroupBasicHandle(t *testing.T) {
	router := New()

	v1 := router.Group("v1", func(c *Context) {})
	assert.Equal(t, "/v1", v1.BasePath())

	login := v1.Group("/login/", func(c *Context) {}, func(c *Context) {})
	assert.Equal(t, "/v1/login/", login.BasePath())

	handler := func(c *Context) {
		c.WriteMessage(&Error{Code: 400, Message: fmt.Sprintf("index %d", c.index)})
		c.Close()
	}
	v1.Handle("/test", handler)
	login.Handle("/test", handler)

	var rc recorder
	router.ServeProto(&rc, &Request{URL: "/v1/login/test"})

	resp, ok := rc.Message.(*Error)
	assert.Equal(t, true, ok, "should return protobuf message.")
	assert.Equal(t, uint32(400), resp.GetCode())
	assert.Equal(t, "index 3", resp.GetMessage())
}

func TestRouterGroupTooManyHandlers(t *testing.T) {
	router := New()
	handlers1 := make([]HandlerFunc, 40)
	router.Use(handlers1...)

	handlers2 := make([]HandlerFunc, 26)
	assert.Panics(t, func() {
		router.Use(handlers2...)
	})
	assert.Panics(t, func() {
		router.Handle("/", handlers2...)
	})
}

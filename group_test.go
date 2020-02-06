package gotham

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRouterHandle(t *testing.T) {
	r := NewRouter()
	r.Handle("pb.Hello", func(c *Context) {})

	assert.Equal(t, 0, len(r.groups))
	assert.Equal(t, "pb.Hello", r.nodes[0].name)

	r.Handle("pb.Bye", func(c *Context) {})

	assert.Equal(t, 0, len(r.groups))
	assert.Equal(t, 2, len(r.nodes))
	assert.Equal(t, "pb.Bye", r.nodes[1].name)

	// test overwrite of the url
	assert.NotPanics(t, func() { r.Handle("pb.Hello", func(c *Context) {}) })
	assert.Equal(t, 2, len(r.nodes))
}

func TestNewRouterMiddlewares(t *testing.T) {
	r := NewRouter()
	var funca HandlerFunc = func(c *Context) {}
	var funcb HandlerFunc = func(c *Context) {}
	r.Use(funca, funcb)
	r.Handle("pb.Hello", func(c *Context) {})
	assert.Equal(t, 1, len(r.nodes))
	assert.Equal(t, 3, len(r.nodes[0].handlers))

	r.Use(funca, funcb)
	assert.NotPanics(t, func() { r.Handle("pb.Hello", func(c *Context) {}) })

	r.Handle("pb.Bye", func(c *Context) {})
	assert.Equal(t, 5, len(r.nodes[1].handlers))
	assert.Equal(t, 2, len(r.nodes))
}

func TestNewRouterGroup(t *testing.T) {
	r := NewRouter()

	r.Use(func(c *Context) {})
	r.Use(func(c *Context) {})

	group := r.Group("test1")
	group.Handle("pb.Hello", func(c *Context) {})
	assert.Equal(t, 3, len(r.nodes[0].handlers))

	group = r.Group("test2")
	group.Handle("pb.Bye", func(c *Context) {})
	assert.Equal(t, 3, len(r.nodes[1].handlers))

	assert.NotPanics(t, func() { group.Handle("pb.Hello", func(c *Context) {}) })
	assert.NotPanics(t, func() { group.Handle("pb.Bye", func(c *Context) {}) })
}

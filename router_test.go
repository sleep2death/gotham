package gotham

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"github.com/sleep2death/gotham/pb"
	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultRouter(t *testing.T) {
	router := Default()
	assert.NotNil(t, router.noRoute)
}

func TestNoRouteWithoutGlobalHandlers(t *testing.T) {
	var middleware0 HandlerFunc = func(c *Context) {}
	var middleware1 HandlerFunc = func(c *Context) {}

	router := New()

	router.NoRoute(middleware0)
	assert.Nil(t, router.handlers)
	assert.Len(t, router.noRoute, 1)
	assert.Len(t, router.allNoRoute, 1)
	compareFunc(t, router.noRoute[0], middleware0)
	compareFunc(t, router.allNoRoute[0], middleware0)

	router.NoRoute(middleware1, middleware0)
	assert.Len(t, router.noRoute, 2)
	assert.Len(t, router.allNoRoute, 2)
	compareFunc(t, router.noRoute[0], middleware1)
	compareFunc(t, router.allNoRoute[0], middleware1)
	compareFunc(t, router.noRoute[1], middleware0)
	compareFunc(t, router.allNoRoute[1], middleware0)

	var middleware2 HandlerFunc = func(c *Context) {}
	router.Use(middleware2)

	compareFunc(t, router.handlers[0], middleware2)

	router.Use(middleware1)
	assert.Len(t, router.handlers, 2)

	compareFunc(t, router.handlers[0], middleware2)
	compareFunc(t, router.handlers[1], middleware1)
}

func compareFunc(t *testing.T, a, b interface{}) {
	sf1 := reflect.ValueOf(a)
	sf2 := reflect.ValueOf(b)
	if sf1.Pointer() != sf2.Pointer() {
		t.Error("different functions")
	}
}

func TestRebuild404Handlers(t *testing.T) {

}

func TestListOfRoutes(t *testing.T) {
	router := New()
	router.Handle("/favicon.ico", handlerTest1)
	router.Handle("/", handlerTest1)
	group := router.Group("/users")
	{
		group.Handle("/id", handlerTest2)
	}
	list := router.Routes()

	assert.Len(t, list, 3)
	assertRoutePresent(t, list, RouteInfo{
		Path:    "/favicon.ico",
		Handler: "^(.*/vendor/)?github.com/sleep2death/gotham.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Path:    "/",
		Handler: "^(.*/vendor/)?github.com/sleep2death/gotham.handlerTest1$",
	})
	assertRoutePresent(t, list, RouteInfo{
		Path:    "/id",
		Handler: "^(.*/vendor/)?github.com/sleep2death/gotham.handlerTest2$",
	})
}

func assertRoutePresent(t *testing.T, gotRoutes RoutesInfo, wantRoute RouteInfo) {
	for _, gotRoute := range gotRoutes {
		if gotRoute.Path == wantRoute.Path {
			assert.Regexp(t, wantRoute.Handler, gotRoute.Handler)
			return
		}
	}
	t.Errorf("route not found: %v", wantRoute)
}

func TestEngineHandleContext(t *testing.T) {
	r := New()
	r.Handle("/", func(c *Context) {
		c.Request.TypeURL = "/redirect"
		r.HandleContext(c)
	})
	v2 := r.Group("/v2")
	v2.Handle("/redirect", func(c *Context) {
		c.Write(&pb.Ping{Message: "redirect"})
	})

	assert.NotPanics(t, func() {
		w := &respRecorder{}
		r.ServeProto(w, &Request{TypeURL: "/"})
		assert.Equal(t, "redirect", w.Message.(*pb.Ping).GetMessage())
	})
}

func handlerTest1(c *Context) {}
func handlerTest2(c *Context) {}

func TestRouterServe(t *testing.T) {
	str := "Ping"

	r := New()
	group := r.Group("pb")
	group.Use(func(ctx *Context) {
		// log.Printf("[middleware]")
		assert.Equal(t, "pb.Ping", ctx.Request.TypeURL)
	})

	group.Handle("pb.Ping", func(ctx *Context) {
		var msg pb.Ping
		proto.Unmarshal(ctx.Request.Data.([]byte), &msg)
		msg.Message = "Pong"
		ctx.Write(&msg)
		// log.Printf("ping message: %s", msg.GetMessage())
	})

	go r.Run(":9001", &ProtobufCodec{})
	time.Sleep(time.Millisecond)

	conn, err := net.Dial("tcp", ":9001")
	if err != nil {
		t.Fatal(err)
	}
	// write request
	// log.Println("write data to server")

	w := newBufioWriter(conn)
	br := newBufioReader(conn)

	// write message frame to server
	msg := &pb.Ping{Message: str}
	WriteFrame(w, msg, &ProtobufCodec{})
	w.Flush()

	time.Sleep(time.Millisecond * 5)
	// read response
	fh, err := ReadFrameHeader(br)
	if err != nil {
		t.Fatal(err)
	}
	req, err := ReadFrameBody(br, fh, &ProtobufCodec{})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "pb.Ping", req.TypeURL)
	resp := &pb.Ping{}
	err = proto.Unmarshal(req.Data.([]byte), resp)
	assert.Equal(t, "Pong", resp.GetMessage())

	// reader := bufio.NewReader(conn)
	// packet, err = reader.Read()
	time.Sleep(time.Millisecond * 5)
}

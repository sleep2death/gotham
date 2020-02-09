package gotham

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/sleep2death/gotham/pb"
)

func BenchmarkOneRoute(B *testing.B) {
	router := New()
	router.Handle("pb.Ping", func(c *Context) {
	})
	runRequest(B, router, "pb.Ping")
}

func BenchmarkWithRecoveryMiddleware(B *testing.B) {
	router := New()
	router.Use(Recovery())
	router.Handle("pb.Ping", func(c *Context) {
	})
	runRequest(B, router, "pb.Ping")
}

func BenchmarkLoggerMiddleware(B *testing.B) {
	router := New()
	router.Use(LoggerWithWriter(new(mockIOWriter)))

	router.Handle("/pb/Ping", func(c *Context) {
	})
	runRequest(B, router, "/pb/Ping")
}

func BenchmarkManyHandlers(B *testing.B) {
	router := New()
	router.Use(Recovery(), LoggerWithWriter(new(mockIOWriter)))
	router.Use(func(c *Context) {})
	router.Use(func(c *Context) {})
	router.Handle("pb.Ping", func(c *Context) {})
	runRequest(B, router, "pb.Ping")
}

func BenchmarkDecodeAndEncode(B *testing.B) {
	router := New()
	router.Handle("pb.Ping", func(c *Context) {
		msg := new(pb.Ping)
		proto.Unmarshal(c.Request.Data(), msg)

		msg.Message = "Pong"
		c.Write(msg)
	})

	runRequest(B, router, "pb.Ping")
}

func BenchmarkOneRouteSet(B *testing.B) {
	router := New()
	router.Handle("pb.Ping", func(c *Context) {
		c.Set("hello", "world")
	})
	runRequest(B, router, "pb.Ping")
}

type mockWriter struct{}

func (mw *mockWriter) Flush() error                  { return nil }
func (mw *mockWriter) Buffered() int                 { return 0 }
func (mw *mockWriter) SetStatus(code int)            {}
func (mw *mockWriter) Status() int                   { return 200 }
func (mw *mockWriter) KeepAlive() bool               { return true }
func (mw *mockWriter) Write(msg proto.Message) error { return nil }

type mockIOWriter struct{}

func (rw *mockIOWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func runRequest(B *testing.B, r *Router, path string) {
	SetMode("release")
	// fake request
	msg := pb.Ping{Message: "Ping"}
	data, _ := proto.Marshal(&msg)
	req := &Request{typeurl: path, data: data}

	w := &mockWriter{}

	B.ReportAllocs()
	B.ResetTimer()

	for i := 0; i < B.N; i++ {
		r.ServeProto(w, req)
	}
}

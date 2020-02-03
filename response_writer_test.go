package gotham

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"testing"

	"github.com/sleep2death/gotham/pb"
	"github.com/stretchr/testify/assert"
)

func TestCreateResponseWriter(t *testing.T) {
	var writer bufio.Writer
	rw := NewResponseWriter(&writer)

	assert.Equal(t, http.StatusOK, rw.Status())
	assert.Equal(t, 0, rw.Buffered())
	assert.Equal(t, true, rw.KeepAlive())
}

func TestResponseWriterWrite(t *testing.T) {
	var b bytes.Buffer
	rw := NewResponseWriter(bufio.NewWriter(&b))

	err := rw.Write(&pb.Ping{Message: "hola"})
	assert.Equal(t, 23, rw.Buffered())
	assert.Equal(t, http.StatusOK, rw.Status())
	// assert.Equal(t, "hola", testWriter.Body.String())
	assert.NoError(t, err)
}

func TestResponseWriterFlush(t *testing.T) {
	addr := "127.0.0.1:9000"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	var handler Handler
	server := &Server{Addr: addr, Handler: handler}
	go server.Serve(ln)
	defer server.Close()

	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
	}

	w := bufio.NewWriter(conn)
	rw := NewResponseWriter(w)
	msg := &pb.Ping{
		Message: "hola",
	}
	rw.Write(msg)
	assert.Equal(t, 23, rw.Buffered())
	err = w.Flush()
	assert.Nil(t, err)
	assert.Equal(t, 0, rw.Buffered())
}

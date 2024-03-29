package gotham

import (
	"bufio"
	"crypto/sha1"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/sleep2death/gotham/pb"
	"github.com/stretchr/testify/assert"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

func TestListenAndServe(t *testing.T) {
	addr := "fataladdr"
	err := ListenAndServe(addr, nil, nil)

	assert.Error(t, err, "missing port in address")

	addr = ""
	err = ListenAndServe(addr, nil, nil)
	assert.EqualError(t, err, "empty address")

	addr = ":9000"
	var handler Handler
	server := &Server{Addr: addr, Handler: handler, Codec: &ProtobufCodec{}}

	// start the server
	go server.ListenAndServe()

	time.Sleep(time.Millisecond * 5)
	server.Shutdown()
	// once shutting down, can not serve again
	err = server.ListenAndServe()
	assert.EqualError(t, err, ErrServerClosed.Error())
}

func TestServe(t *testing.T) {
	addr := "127.0.0.1:9000"
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	var handler Handler
	server := &Server{Addr: addr, Handler: handler, Codec: &ProtobufCodec{}}
	// conn will close, if no message was read in 50ms
	server.ReadTimeout = time.Millisecond * 50
	go server.Serve(ln)
	defer server.Close()

	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
	}

	// write to the server, so activeConn should track the conn
	conn.Write([]byte("test"))
	time.Sleep(time.Millisecond)

	assert.Equal(t, addr, conn.RemoteAddr().String())

	server.mu.Lock()
	assert.Equal(t, 1, len(server.activeConn))
	server.mu.Unlock()

	// sleep 100ms, so the conn will be timeout, and closed
	time.Sleep(time.Millisecond * 100)

	server.mu.Lock()
	assert.Equal(t, 0, len(server.activeConn))
	server.mu.Unlock()

	// client can't read anymore from closed conn
	var replyBuffer = make([]byte, 256)
	_, err = conn.Read(replyBuffer)
	assert.EqualError(t, err, "EOF")

	// connect to server
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
	}

	w := bufio.NewWriter(conn)
	msg := &pb.Ping{
		Message: "Ping",
	}
	WriteFrame(w, msg, &ProtobufCodec{})
	w.Flush()

	// sleep 100ms, so the conn will be idle, and closed
	time.Sleep(time.Millisecond * 100)
	server.mu.Lock()
	assert.Equal(t, 0, len(server.activeConn))
	server.mu.Unlock()

	time.Sleep(time.Millisecond * 10)
}

type tHandler struct {
}

func (rr *tHandler) ServeProto(w ResponseWriter, req *Request) {
	switch req.TypeURL {
	case "pb.Ping":
		var msg pb.Ping
		msg.Message = "Pong"
		w.Write(&msg)
	case "pb.Error":
		var msg pb.Error
		msg.Code = 400
		msg.Message = "Pong Error"

		w.Write(&msg)
		w.(*responseWriter).keepAlive = false
	default:
		// log.Println("no url handler found")
		panic("no url handler found")
	}
}

func TestReadWriteData(t *testing.T) {
	addr := ":9000"
	server := &Server{Addr: addr, Handler: &tHandler{}, Codec: &ProtobufCodec{}}
	go server.ListenAndServe()
	defer server.Close()

	time.Sleep(time.Millisecond * 5)
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	ping := &pb.Ping{
		Message: "Ping",
	}
	content, _ := proto.Marshal(ping)

	any := &any.Any{
		TypeUrl: "pb.Ping",
		Value:   content,
	}

	payload, _ := proto.Marshal(any)

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	// write two frames at once
	WriteData(w, payload)
	WriteData(w, payload)
	w.Flush()

	// wait for response
	time.Sleep(time.Millisecond * 5)
	var pong pb.Ping
	// read one
	res, err := ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())
	// read two
	res, err = ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())

	time.Sleep(time.Millisecond * 5)

	// make the payload manually
	var flags Flags
	flags |= FlagFrameAck
	length := len(payload)

	header := [frameHeaderLen]byte{
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
		byte(FrameData),
		byte(flags),
	}

	// test incomplete header
	// write the broken header first
	w.Write(header[:3])
	w.Flush()
	// then wait a little while, write the left...
	time.Sleep(time.Millisecond * 5)
	wbuf := append(header[3:], payload...)
	w.Write(wbuf)
	w.Flush()

	time.Sleep(time.Millisecond * 5)
	res, err = ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())
	// test incomplete body
	// write the broken body first
	wbuf = append(header[:frameHeaderLen], payload[:3]...)
	w.Write(wbuf)
	w.Flush()
	// then wait a little while, write the left...
	time.Sleep(time.Millisecond * 5)
	wbuf = append(payload[3:])
	w.Write(wbuf)
	w.Flush()
	time.Sleep(time.Millisecond * 5)
	res, err = ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())
}

func TestWriteFrame(t *testing.T) {
	addr := ":9000"
	server := &Server{Addr: addr, Handler: &tHandler{}, Codec: &ProtobufCodec{}}
	go server.ListenAndServe()
	defer server.Close()

	time.Sleep(time.Millisecond * 5)
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	w := newBufioWriter(conn)
	r := newBufioReader(conn)

	WriteFrame(w, &pb.Ping{Message: "Ping"}, &ProtobufCodec{})
	w.Flush()

	time.Sleep(time.Millisecond * 5)
	var pong pb.Ping
	res, _ := ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())

	putBufioReader(r)
	putBufioWriter(w)

	w = newBufioWriter(conn)
	r = newBufioReader(conn)

	WriteFrame(w, &pb.Ping{Message: "Ping"}, &ProtobufCodec{})
	w.Flush()

	res, _ = ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())
}

func TestErrorFrame(t *testing.T) {
	addr := ":9000"
	server := &Server{Addr: addr, Handler: &tHandler{}, Codec: &ProtobufCodec{}}
	go server.ListenAndServe()
	defer server.Close()

	time.Sleep(time.Millisecond * 5)
	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	w := newBufioWriter(conn)
	r := newBufioReader(conn)

	WriteFrame(w, &pb.Error{Code: 400, Message: "Ping Error"}, &ProtobufCodec{})
	w.Flush()

	// server close the conn because of the error
	time.Sleep(time.Millisecond * 5)
	server.mu.Lock()
	assert.Equal(t, 0, len(server.activeConn))
	server.mu.Unlock()

	// still get the error response
	var msg pb.Error
	res, _ := ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &msg)
	assert.Equal(t, "Pong Error", msg.GetMessage())

}

func TestServerShutDown(t *testing.T) {
	addr := ":9000"
	server := &Server{Addr: addr, Handler: &tHandler{}, Codec: &ProtobufCodec{}}
	go server.ListenAndServe()

	time.Sleep(time.Millisecond * 5)
	// connect to server
	for i := 0; i < 50; i++ {
		_, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond * 5)
	}

	server.Shutdown()
}

func TestServerKCP(t *testing.T) {
	addr := "127.0.0.1:9000"

	key := pbkdf2.Key([]byte("demo pass"), []byte("demo salt"), 1024, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)

	ln, err := kcp.ListenWithOptions(addr, block, 10, 3)
	if err != nil {
		t.Fatal(err)
	}

	// var handler Handler
	server := &Server{Addr: addr, Handler: &tHandler{}, Codec: &ProtobufCodec{}}

	// conn will close, if no message was read in 500ms
	server.ReadTimeout = time.Millisecond * 500

	go server.Serve(ln)
	defer server.Close()

	time.Sleep(time.Millisecond)
	// connect to server
	conn, err := kcp.DialWithOptions(addr, block, 10, 3)
	if err != nil {
		t.Error(err)
	}

	t.Log("kcp connected")

	ping := &pb.Ping{
		Message: "Ping",
	}
	content, _ := proto.Marshal(ping)

	any := &any.Any{
		TypeUrl: "pb.Ping",
		Value:   content,
	}

	payload, _ := proto.Marshal(any)

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	// write two frames at once
	WriteData(w, payload)
	WriteData(w, payload)
	w.Flush()

	// wait for response
	time.Sleep(time.Millisecond * 5)
	var pong pb.Ping
	// read one
	res, err := ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())
	// read two
	res, err = ReadFrame(r, &ProtobufCodec{})
	proto.Unmarshal(res.Data.([]byte), &pong)
	assert.Equal(t, "Pong", pong.GetMessage())

	t.Log("kcp read")

	time.Sleep(time.Millisecond * 5)
}

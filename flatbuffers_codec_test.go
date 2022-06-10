package gotham

import (
	"bufio"
	"fmt"
	"net"
	"testing"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/sleep2death/gotham/fbs"

	"github.com/stretchr/testify/require"
)

func TestFlatbuffersMarshal(t *testing.T) {
	ping := &fbs.PingT{
		Timestamp: time.Now().Unix(),
	}

	msg := &fbs.MessageT{}

	// add ping data to message
	msg.Data = &fbs.AnyT{Type: fbs.AnyPing, Value: ping}

	builder := flatbuffers.NewBuilder(0)
	builder.Finish(msg.Pack(builder))

	fbc := &FlatbuffersCodec{}
	req := &Request{}

	err := fbc.Unmarshal(builder.FinishedBytes(), req)

	require.NoError(t, err)
	require.Equal(t, "Ping", req.TypeURL)
}

func TestFlatbuffersUnmarshale(t *testing.T) {
	builder := flatbuffers.NewBuilder(0)

	now := time.Now().Unix()

	fbs.PongStart(builder)
	fbs.PongAddTimestamp(builder, now)
	pong := fbs.PongEnd(builder)

	fbs.MessageStart(builder)
	fbs.MessageAddDataType(builder, fbs.AnyPong)
	fbs.MessageAddData(builder, pong)
	msg := fbs.MessageEnd(builder)

	builder.Finish(msg)

	fbc := &FlatbuffersCodec{}
	req := &Request{}

	err := fbc.Unmarshal(builder.FinishedBytes(), req)

	require.NoError(t, err)
	require.Equal(t, "Pong", req.TypeURL)

	if d, ok := req.Data.(*fbs.Pong); ok {
		require.Equal(t, now, d.Timestamp())
	}
}

func TestFlatbuffersCodec(t *testing.T) {

	addr := ":9000"
	server := &Server{Addr: addr, Handler: &ttHandler{}, Codec: &FlatbuffersCodec{}}
	go server.ListenAndServe()
	t.Log("start server...")
	defer server.Close()

	time.Sleep(time.Millisecond * 5)

	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
	}

	now := time.Now().Unix()
	ping := &fbs.PingT{
		Timestamp: now,
	}

	msg := &fbs.MessageT{}

	// add ping data to message
	msg.Data = &fbs.AnyT{Type: fbs.AnyPing, Value: ping}

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	err = WriteFrame(w, msg, &FlatbuffersCodec{})
	if err != nil {
		t.Fatal(err)
	}

	w.Flush()

	time.Sleep(time.Millisecond * 10)

	t.Log("start reading...")
	req, err := ReadFrame(r, &FlatbuffersCodec{})
	require.Equal(t, "Pong", req.TypeURL)
	// require.Greater(t, req.Data.(*fbs.AnyT).Value.(*fbs.PongT).Timestamp, now)
}

type ttHandler struct {
}

func (rr *ttHandler) ServeProto(w ResponseWriter, req *Request) {
	switch req.TypeURL {
	case "Ping":
		now := time.Now().Unix()
		pong := &fbs.PongT{
			Timestamp: now,
		}
		msg := &fbs.MessageT{}
		// add ping data to message
		msg.Data = &fbs.AnyT{Type: fbs.AnyPong, Value: pong}
		err := w.Write(msg)
		if err != nil {
			fmt.Println(err)
		}

		w.(*responseWriter).keepAlive = false
	case "Pong":
		ping := &fbs.PingT{
			Timestamp: time.Now().Unix(),
		}
		msg := &fbs.MessageT{}
		// add ping data to message
		msg.Data = &fbs.AnyT{Type: fbs.AnyPing, Value: ping}
		err := w.Write(&msg)
		if err != nil {
			fmt.Println(err)
		}
		w.(*responseWriter).keepAlive = false
	default:
		// log.Println("no url handler found")
		panic("no url handler found")
	}
}

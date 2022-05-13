package gotham

import (
	"bufio"
	"net"
	"testing"
	"time"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/sleep2death/gotham/fbs"

	"github.com/stretchr/testify/require"
)

func TestFlatbuffersMarshal(t *testing.T) {
	ping := &fbs.PingT{
		TimeStamp: time.Now().Unix(),
	}

	msg := &fbs.MessageT{}
	msg.Url = "ping"

	// add ping data to message
	msg.Data = &fbs.AnyT{Type: fbs.AnyPing, Value: ping}

	builder := flatbuffers.NewBuilder(0)
	builder.Finish(msg.Pack(builder))

	fbc := &FlatbuffersCodec{}
	req := &Request{}

	err := fbc.Unmarshal(builder.FinishedBytes(), req)

	require.NoError(t, err)
	require.Equal(t, "ping", req.TypeURL)
}

func TestFlatbuffersUnmarshale(t *testing.T) {
	builder := flatbuffers.NewBuilder(0)

	url := builder.CreateString("pong")
	now := time.Now().Unix()

	fbs.PongStart(builder)
	fbs.PongAddTimeStamp(builder, now)
	pong := fbs.PongEnd(builder)

	fbs.MessageStart(builder)
	fbs.MessageAddUrl(builder, url)
	fbs.MessageAddDataType(builder, fbs.AnyPong)
	fbs.MessageAddData(builder, pong)
	msg := fbs.MessageEnd(builder)

	builder.Finish(msg)

	fbc := &FlatbuffersCodec{}
	req := &Request{}

	err := fbc.Unmarshal(builder.FinishedBytes(), req)

	require.NoError(t, err)
	require.Equal(t, "pong", req.TypeURL)

	if d, ok := req.Data.(*fbs.Pong); ok {
		require.Equal(t, now, d.TimeStamp())
	}
}

func TestFlatbuffersCodec(t *testing.T) {
	addr := ":9000"
	server := &Server{Addr: addr, Handler: &ttHandler{}, Codec: &FlatbuffersCodec{}}
	go server.ListenAndServe()
	defer server.Close()

	time.Sleep(time.Millisecond * 5)

	// connect to server
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Error(err)
	}

	now := time.Now().Unix()
	ping := &fbs.PingT{
		TimeStamp: now,
	}

	msg := &fbs.MessageT{}
	msg.Url = "ping"

	// add ping data to message
	msg.Data = &fbs.AnyT{Type: fbs.AnyPing, Value: ping}

	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	WriteFrame(w, msg, &ProtobufCodec{})
	w.Flush()

	time.Sleep(time.Millisecond * 10)

	req, err := ReadFrame(r, &FlatbuffersCodec{})
	require.Equal(t, "pong", req.TypeURL)
	require.Greater(t, req.Data.(*fbs.PongT).TimeStamp, now)

	time.Sleep(time.Millisecond * 10)
}

type ttHandler struct {
}

func (rr *ttHandler) ServeProto(w ResponseWriter, req *Request) {
	switch req.TypeURL {
	case "ping":
		var msg fbs.PongT
		msg.TimeStamp = time.Now().Unix()
		w.Write(&msg)
	case "pong":
		var msg fbs.PingT
		msg.TimeStamp = time.Now().Unix()

		w.Write(&msg)
		w.(*responseWriter).keepAlive = false
	default:
		// log.Println("no url handler found")
		panic("no url handler found")
	}
}

package gotham

import (
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

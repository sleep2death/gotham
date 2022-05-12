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

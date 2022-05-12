package gotham

import (
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	fbs "github.com/sleep2death/gotham/fbs"
)

type FlatBuffersCodec struct {
}

func (pc *FlatBuffersCodec) Unmarshal(data []byte, req *Request) error {
	msgt := fbs.GetRootAsMessage(data, 0).UnPack()
	req.TypeURL = msgt.Url
	req.Data = msgt.Data
	return nil
}

func (pc *FlatBuffersCodec) Marshal(data interface{}) ([]byte, error) {
	if m, ok := data.(fbs.MessageT); ok {
		builder := flatbuffers.NewBuilder(0)
		builder.Finish(m.Pack(builder))

		return builder.FinishedBytes(), nil
	}
	return nil, fmt.Errorf("not a flatbuffers message: %v", data)
}

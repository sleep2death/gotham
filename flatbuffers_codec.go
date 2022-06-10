package gotham

import (
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	fbs "github.com/sleep2death/gotham/fbs"
)

type FlatbuffersCodec struct {
}

func (pc *FlatbuffersCodec) Unmarshal(data []byte, req *Request) error {
	// fmt.Println("unmarhsal")
	msgt := fbs.GetRootAsMessage(data, 0).UnPack()
	req.TypeURL = msgt.Data.Type.String()
	req.Data = msgt.Data
	return nil
}

func (pc *FlatbuffersCodec) Marshal(data interface{}) ([]byte, error) {
	// fmt.Println("marshal")
	if m, ok := data.(*fbs.MessageT); ok {
		builder := flatbuffers.NewBuilder(0)
		builder.Finish(m.Pack(builder))

		return builder.FinishedBytes(), nil
	}
	return nil, fmt.Errorf("not a flatbuffers message: %v", data)
}

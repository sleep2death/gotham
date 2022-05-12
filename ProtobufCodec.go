package gotham

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

type ProtobufCodec struct {
}

func (pc *ProtobufCodec) Unmarshal(data []byte, req *Request) error {
	var msg any.Any
	err := proto.Unmarshal(data, &msg)

	if err != nil {
		return err
	}

	req.Data = msg.GetValue()
	req.TypeURL = msg.GetTypeUrl()
	return nil
}

func (pc *ProtobufCodec) Marshal(data interface{}) ([]byte, error) {
	if m, ok := data.(proto.Message); ok {
		// marshal the payload pb
		buf, err := proto.Marshal(m)
		if err != nil {
			return nil, err
		}

		// transfer dot to slash
		url := proto.MessageName(m)
		// wrap it to any pb
		anyMsg := &any.Any{
			TypeUrl: url,
			Value:   buf,
		}
		// marshal the any pb
		buf, err = proto.Marshal(anyMsg)
		if err != nil {
			return nil, err
		}

		return buf, nil
	}

  return nil, fmt.Errorf("not a prototype message: %v", data)
}

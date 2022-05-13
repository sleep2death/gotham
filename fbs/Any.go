// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package fbs

import (
	"strconv"

	flatbuffers "github.com/google/flatbuffers/go"
)

type Any byte

const (
	AnyNONE  Any = 0
	AnyPing  Any = 1
	AnyPong  Any = 2
	AnyError Any = 3
)

var EnumNamesAny = map[Any]string{
	AnyNONE:  "NONE",
	AnyPing:  "Ping",
	AnyPong:  "Pong",
	AnyError: "Error",
}

var EnumValuesAny = map[string]Any{
	"NONE":  AnyNONE,
	"Ping":  AnyPing,
	"Pong":  AnyPong,
	"Error": AnyError,
}

func (v Any) String() string {
	if s, ok := EnumNamesAny[v]; ok {
		return s
	}
	return "Any(" + strconv.FormatInt(int64(v), 10) + ")"
}

type AnyT struct {
	Type Any
	Value interface{}
}

func (t *AnyT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil {
		return 0
	}
	switch t.Type {
	case AnyPing:
		return t.Value.(*PingT).Pack(builder)
	case AnyPong:
		return t.Value.(*PongT).Pack(builder)
	case AnyError:
		return t.Value.(*ErrorT).Pack(builder)
	}
	return 0
}

func (rcv Any) UnPack(table flatbuffers.Table) *AnyT {
	switch rcv {
	case AnyPing:
		x := Ping{_tab: table}
		return &AnyT{ Type: AnyPing, Value: x.UnPack() }
	case AnyPong:
		x := Pong{_tab: table}
		return &AnyT{ Type: AnyPong, Value: x.UnPack() }
	case AnyError:
		x := Error{_tab: table}
		return &AnyT{ Type: AnyError, Value: x.UnPack() }
	}
	return nil
}
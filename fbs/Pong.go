// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package fbs

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type PongT struct {
	Timestamp int64
}

func (t *PongT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil { return 0 }
	PongStart(builder)
	PongAddTimestamp(builder, t.Timestamp)
	return PongEnd(builder)
}

func (rcv *Pong) UnPackTo(t *PongT) {
	t.Timestamp = rcv.Timestamp()
}

func (rcv *Pong) UnPack() *PongT {
	if rcv == nil { return nil }
	t := &PongT{}
	rcv.UnPackTo(t)
	return t
}

type Pong struct {
	_tab flatbuffers.Table
}

func GetRootAsPong(buf []byte, offset flatbuffers.UOffsetT) *Pong {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Pong{}
	x.Init(buf, n+offset)
	return x
}

func GetSizePrefixedRootAsPong(buf []byte, offset flatbuffers.UOffsetT) *Pong {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &Pong{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func (rcv *Pong) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Pong) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *Pong) Timestamp() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Pong) MutateTimestamp(n int64) bool {
	return rcv._tab.MutateInt64Slot(4, n)
}

func PongStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func PongAddTimestamp(builder *flatbuffers.Builder, timestamp int64) {
	builder.PrependInt64Slot(0, timestamp, 0)
}
func PongEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}

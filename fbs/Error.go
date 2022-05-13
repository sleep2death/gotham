// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package fbs

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type ErrorT struct {
	Message string
	TimeStamp int64
}

func (t *ErrorT) Pack(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	if t == nil { return 0 }
	messageOffset := builder.CreateString(t.Message)
	ErrorStart(builder)
	ErrorAddMessage(builder, messageOffset)
	ErrorAddTimeStamp(builder, t.TimeStamp)
	return ErrorEnd(builder)
}

func (rcv *Error) UnPackTo(t *ErrorT) {
	t.Message = string(rcv.Message())
	t.TimeStamp = rcv.TimeStamp()
}

func (rcv *Error) UnPack() *ErrorT {
	if rcv == nil { return nil }
	t := &ErrorT{}
	rcv.UnPackTo(t)
	return t
}

type Error struct {
	_tab flatbuffers.Table
}

func GetRootAsError(buf []byte, offset flatbuffers.UOffsetT) *Error {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &Error{}
	x.Init(buf, n+offset)
	return x
}

func GetSizePrefixedRootAsError(buf []byte, offset flatbuffers.UOffsetT) *Error {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &Error{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func (rcv *Error) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *Error) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *Error) Message() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *Error) TimeStamp() int64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return rcv._tab.GetInt64(o + rcv._tab.Pos)
	}
	return 0
}

func (rcv *Error) MutateTimeStamp(n int64) bool {
	return rcv._tab.MutateInt64Slot(6, n)
}

func ErrorStart(builder *flatbuffers.Builder) {
	builder.StartObject(2)
}
func ErrorAddMessage(builder *flatbuffers.Builder, message flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(message), 0)
}
func ErrorAddTimeStamp(builder *flatbuffers.Builder, timeStamp int64) {
	builder.PrependInt64Slot(1, timeStamp, 0)
}
func ErrorEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
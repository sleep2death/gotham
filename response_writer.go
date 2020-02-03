package gotham

import (
	"bufio"
	"io"
	"net/http"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

// RespWriter interface is used by a handler to construct an protobuf response.
type RespWriter interface {
	// Set the response status code of the current request.
	SetStatus(statusCode int)

	// Returns the response status code of the current request.
	Status() int

	// Write the protobuf into sending buffer.
	Write(message proto.Message) error

	// Returns the number of bytes already written into the response.
	// See Written()
	Size() int

	// Returns true if the response was already written.
	Written() bool
}

type respWriter struct {
	writer *bufio.Writer
	size   int
	status int
}

func (rw *respWriter) SetStatus(code int) {
	rw.status = code
}

func (rw *respWriter) Status() int {
	return rw.status
}

func (rw *respWriter) Size() int {
	if rw.writer != nil {
		return rw.writer.Buffered()
	}
	return 0
}

func (rw *respWriter) Written() bool {
	return rw.Size() > 0
}

func (rw *respWriter) Write(message proto.Message) error {
	return WriteFrame(rw.writer, message)
}

// WriteFrame with given url
func WriteFrame(w io.Writer, message proto.Message) error {
	// marshal the payload pb
	buf, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	// transfer dot to slash
	url := "/" + strings.Replace(proto.MessageName(message), ".", "/", -1)
	// wrap it to any pb
	any := &types.Any{
		TypeUrl: url,
		Value:   buf,
	}
	// marshal the any pb
	buf, err = any.Marshal()
	if err != nil {
		return err
	}
	// write the frame head
	return WriteData(w, buf)
}

// WriteData with the payload.
func WriteData(w io.Writer, data []byte) (err error) {
	var flags Flags
	// flags |= FlagDataEndStream
	flags |= FlagFrameAck

	length := len(data)
	if length >= (1 << 24) {
		return ErrFrameTooLarge
	}

	header := [frameHeaderLen]byte{
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
		byte(FrameData),
		byte(flags),
	}
	wbuf := append(header[:frameHeaderLen], data...)

	n, err := w.Write(wbuf)

	if err == nil && n != len(wbuf) {
		err = io.ErrShortWrite
	}

	return err
}

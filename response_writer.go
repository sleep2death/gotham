package gotham

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
)

const (
	noWritten        = -1
	defaultKeepAlive = true
	defaultStatus    = http.StatusOK
)

var (
	ErrNotFlusher = errors.New("this writer is not a flusher")
)

type BufWriter interface {
	// Write the protobuf into sending buffer.
	Write(message proto.Message) error

	// Returns the number of bytes already written into the response.
	// See Written()
	Size() int

	// Flush writes any buffered data to the underlying io.Writer.
	Flush() error
}

// RespWriter interface is used by a handler to construct an protobuf response.
type RespWriter interface {
	BufWriter

	// Set the response status code of the current request.
	SetStatus(statusCode int)

	// Returns the response status code of the current request.
	Status() int

	// Returns false if the server should close the connection after flush the data.
	KeepAlive() bool
}

type respWriter struct {
	writer    io.Writer
	status    int
	keepAlive bool
}

func NewRespWriter(w io.Writer) *respWriter {
	rw := &respWriter{}
	rw.writer = w
	rw.keepAlive = true
	rw.status = defaultStatus
	return rw
}

func (rw *respWriter) SetStatus(code int) {
	rw.status = code
}

func (rw *respWriter) Status() int {
	return rw.status
}

func (rw *respWriter) Size() int {
	if w, ok := rw.writer.(BufWriter); ok {
		return w.Size()
	}
	return noWritten
}

func (rw *respWriter) Flush() error {
	if w, ok := rw.writer.(BufWriter); ok {
		return w.Flush()
	}
	return ErrNotFlusher
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

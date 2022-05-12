package gotham

import (
	"errors"
	"io"
	"net/http"
)

const (
	noWritten        = 0
	defaultKeepAlive = true
	defaultStatus    = http.StatusOK
)

var (
	ErrNotFlusher = errors.New("this writer is not a flusher")
)

type BufFlusher interface {
	// Flush writes any buffered data to the underlying io.Writer.
	Flush() error

	// Returns the number of bytes already written into the response.
	// See Written()
	Buffered() int
}

// ResponseWriter interface is used by a handler to construct an protobuf response.
type ResponseWriter interface {
	BufFlusher

	// Set the response status code of the current request.
	SetStatus(statusCode int)

	// Returns the response status code of the current request.
	Status() int

	// Returns false if the server should close the connection after flush the data.
	KeepAlive() bool

	// Returns false if the server should close the connection after flush the data.
	SetKeepAlive(value bool)

	// Write the data into sending buffer.
	Write(data interface{}) error
}

// responseWriter implements interface ResponseWriter
type responseWriter struct {
	writer    io.Writer
	status    int
	keepAlive bool
	codec     Codec
}

func NewResponseWriter(w io.Writer, c Codec) *responseWriter {
	rw := &responseWriter{}
	rw.writer = w
	rw.keepAlive = true
	rw.status = defaultStatus
	rw.codec = c
	return rw
}

func (rw *responseWriter) SetStatus(code int) {
	rw.status = code
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) KeepAlive() bool {
	return rw.keepAlive
}

func (rw *responseWriter) SetKeepAlive(value bool) {
	rw.keepAlive = value
}

func (rw *responseWriter) Buffered() int {
	if w, ok := rw.writer.(BufFlusher); ok {
		return w.Buffered()
	}
	return noWritten
}

func (rw *responseWriter) Flush() error {
	if w, ok := rw.writer.(BufFlusher); ok {
		return w.Flush()
	}
	return ErrNotFlusher
}

func (rw *responseWriter) Write(data interface{}) error {
	return WriteFrame(rw.writer, data, rw.codec)
}

type respRecorder struct {
	responseWriter
	Message interface{}
	codec   Codec
}

func (rr *respRecorder) Write(data interface{}) error {
	rr.Message = data

	if rr.writer != nil {
		return WriteFrame(rr.writer, data, rr.codec)
	}
	return nil
}

// WriteFrame with given url
func WriteFrame(w io.Writer, data interface{}, codec Codec) error {
	// marshal the payload pb
	buf, err := codec.Marshal(data)
	if err != nil {
		return err
	}

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

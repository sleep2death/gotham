package gotham

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicClean(t *testing.T) {
	buffer := new(bytes.Buffer)
	router := New()
	router.Use(RecoveryWithWriter(buffer))
	router.Handle("/recovery", func(c *Context) {
		c.AbortWithStatus(http.StatusBadRequest)
		panic("Oops, Houston, we have a problem")
	})
	// RUN
	w := performRequest("/recovery", router)
	// Recovery will rewrite the status code to internal-server-error.
	assert.Equal(t, http.StatusInternalServerError, w.Status())
}

// TestPanicInHandler assert that panic has been recovered.
func TestPanicInHandler(t *testing.T) {
	buffer := new(bytes.Buffer)
	router := New()
	router.Use(RecoveryWithWriter(buffer))
	router.Handle("/recovery", func(_ *Context) {
		panic("Oops, Houston, we have a problem")
	})
	// RUN
	w := performRequest("/recovery", router)
	// TEST
	assert.Equal(t, http.StatusInternalServerError, w.Status())
	assert.Contains(t, buffer.String(), "panic recovered")
	assert.Contains(t, buffer.String(), "Oops, Houston, we have a problem")
	assert.Contains(t, buffer.String(), "TestPanicInHandler")
}

// TestPanicWithAbort assert that panic has been recovered even if context.Abort was used.
func TestPanicWithAbort(t *testing.T) {
	router := New()
	router.Use(RecoveryWithWriter(nil))
	router.Handle("/recovery", func(c *Context) {
		c.AbortWithStatus(http.StatusBadRequest)
		panic("Oupps, Houston, we have a problem")
	})
	// RUN
	w := performRequest("/recovery", router)
	// TEST
	assert.Equal(t, http.StatusInternalServerError, w.Status())
}

func TestSource(t *testing.T) {
	bs := source(nil, 0)
	assert.Equal(t, []byte("???"), bs)

	in := [][]byte{
		[]byte("Hello world."),
		[]byte("Hi, gin.."),
	}
	bs = source(in, 10)
	assert.Equal(t, []byte("???"), bs)

	bs = source(in, 1)
	assert.Equal(t, []byte("Hello world."), bs)
}

func TestFunction(t *testing.T) {
	bs := function(1)
	assert.Equal(t, []byte("???"), bs)
}

// TestPanicWithBrokenPipe asserts that recovery specifically
// handles
// writing responses to broken pipes
func TestPanicWithBrokenPipe(t *testing.T) {
	const expectCode = 204

	expectMsgs := map[syscall.Errno]string{
		syscall.EPIPE:      "broken pipe",
		syscall.ECONNRESET: "connection reset by peer",
	}

	for errno, expectMsg := range expectMsgs {
		t.Run(expectMsg, func(t *testing.T) {

			var buf bytes.Buffer

			router := New()
			router.Use(RecoveryWithWriter(&buf))
			router.Handle("/recovery", func(c *Context) {
				// Start
				// writing
				// response
				// c.Header("X-Test", "Value")
				c.Writer.SetStatus(expectCode)

				// Oops.
				// Client
				// connection
				// closed
				e := &net.OpError{Err: &os.SyscallError{Err: errno}}
				panic(e)
			})
			// RUN
			w := performRequest("/recovery", router)
			// TEST
			assert.Equal(t, expectCode, w.Status())
			assert.Contains(t, strings.ToLower(buf.String()), expectMsg)
		})
	}
}

package gotham

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func performRequest(url string, r *Router) *respRecorder {
	b := new(bytes.Buffer)
	rr := &respRecorder{}
	rr.writer = b
	rr.status = http.StatusOK
	r.ServeProto(rr, &Request{URL: url})
	return rr
}

func TestLogger(t *testing.T) {
	buffer := new(bytes.Buffer)
	router := New()
	router.NoRoute(DefaultNoRouteHandler)
	router.Use(LoggerWithWriter(buffer))
	router.Handle("/example", func(c *Context) {})

	performRequest("/example", router)
	// performRequest(router, "GET", "/notfound")
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "/example")

	buffer.Reset()
	performRequest("/notfound", router)
	assert.Contains(t, buffer.String(), "404")
	assert.Contains(t, buffer.String(), "/notfound")
}

func TestLoggerWithConfig(t *testing.T) {
	buffer := new(bytes.Buffer)
	router := New()
	router.NoRoute(DefaultNoRouteHandler)
	router.Use(LoggerWithConfig(LoggerConfig{Output: buffer}))
	router.Handle("/example", func(c *Context) {})

	performRequest("/example", router)

	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "/example")

	// I wrote these first (extending the above) but then realized they are more
	// like integration tests because they test the whole logging process rather
	// than individual functions.  Im not sure where these should go.
	buffer.Reset()
	performRequest("/notfound", router)
	assert.Contains(t, buffer.String(), "404")
	assert.Contains(t, buffer.String(), "/notfound")
}

func TestLoggerWithFormatter(t *testing.T) {
	buffer := new(bytes.Buffer)

	d := DefaultWriter
	DefaultWriter = buffer
	defer func() {
		DefaultWriter = d
	}()

	router := New()
	router.Use(LoggerWithFormatter(func(param LogFormatterParams) string {
		return fmt.Sprintf("[FORMATTER TEST] %v | %3d | %13v | %15s | %s\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Path,
			param.ErrorMessage,
		)
	}))
	router.Handle("/example", func(c *Context) {})
	performRequest("/example", router)

	// output test
	assert.Contains(t, buffer.String(), "[FORMATTER TEST]")
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "/example")
}

func TestLoggerWithConfigFormatting(t *testing.T) {
	var gotParam LogFormatterParams
	var gotKeys map[string]interface{}
	buffer := new(bytes.Buffer)

	router := New()
	router.Use(LoggerWithConfig(LoggerConfig{
		Output: buffer,
		Formatter: func(param LogFormatterParams) string {
			// for assert test
			gotParam = param

			return fmt.Sprintf("[FORMATTER TEST] %v | %3d | %13v | %15s | %s\n%s",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.StatusCode,
				param.Latency,
				param.ClientIP,
				param.Path,
				param.ErrorMessage,
			)
		},
	}))
	router.Handle("/example", func(c *Context) {
		gotKeys = c.Keys
	})

	performRequest("/example", router)

	// output test
	assert.Contains(t, buffer.String(), "[FORMATTER TEST]")
	assert.Contains(t, buffer.String(), "200")
	assert.Contains(t, buffer.String(), "/example")

	// LogFormatterParams test
	assert.NotNil(t, gotParam.Request)
	assert.NotEmpty(t, gotParam.TimeStamp)
	assert.Equal(t, 200, gotParam.StatusCode)
	assert.NotEmpty(t, gotParam.Latency)
	assert.Equal(t, "0.0.0.0", gotParam.ClientIP)
	assert.Equal(t, "/example", gotParam.Path)
	assert.Empty(t, gotParam.ErrorMessage)
	assert.Equal(t, gotKeys, gotParam.Keys)
}

func TestDefaultLogFormatter(t *testing.T) {
	timeStamp := time.Unix(1544173902, 0).UTC()

	termFalseParam := LogFormatterParams{
		TimeStamp:    timeStamp,
		StatusCode:   200,
		Latency:      time.Second * 5,
		ClientIP:     "20.20.20.20",
		Path:         "/",
		ErrorMessage: "",
		isTerm:       false,
	}

	termTrueParam := LogFormatterParams{
		TimeStamp:    timeStamp,
		StatusCode:   200,
		Latency:      time.Second * 5,
		ClientIP:     "20.20.20.20",
		Path:         "/",
		ErrorMessage: "",
		isTerm:       true,
	}
	termTrueLongDurationParam := LogFormatterParams{
		TimeStamp:    timeStamp,
		StatusCode:   200,
		Latency:      time.Millisecond * 9876543210,
		ClientIP:     "20.20.20.20",
		Path:         "/",
		ErrorMessage: "",
		isTerm:       true,
	}

	termFalseLongDurationParam := LogFormatterParams{
		TimeStamp:    timeStamp,
		StatusCode:   200,
		Latency:      time.Millisecond * 9876543210,
		ClientIP:     "20.20.20.20",
		Path:         "/",
		ErrorMessage: "",
		isTerm:       false,
	}

	assert.Equal(t, "[GOTHAM] 2018/12/07 - 09:11:42 | 200 |            5s |     20.20.20.20 | /\n", defaultLogFormatter(termFalseParam))
	assert.Equal(t, "[GOTHAM] 2018/12/07 - 09:11:42 | 200 |    2743h29m3s |     20.20.20.20 | /\n", defaultLogFormatter(termFalseLongDurationParam))

	assert.Equal(t, "[GOTHAM] 2018/12/07 - 09:11:42 |\x1b[97;42m 200 \x1b[0m|            5s |     20.20.20.20 |\x1b[0m /\n", defaultLogFormatter(termTrueParam))
	assert.Equal(t, "[GOTHAM] 2018/12/07 - 09:11:42 |\x1b[97;42m 200 \x1b[0m|    2743h29m3s |     20.20.20.20 |\x1b[0m /\n", defaultLogFormatter(termTrueLongDurationParam))

}

func TestColorForStatus(t *testing.T) {
	colorForStatus := func(code int) string {
		p := LogFormatterParams{
			StatusCode: code,
		}
		return p.StatusCodeColor()
	}

	assert.Equal(t, green, colorForStatus(http.StatusOK), "2xx should be green")
	assert.Equal(t, white, colorForStatus(http.StatusMovedPermanently), "3xx should be white")
	assert.Equal(t, yellow, colorForStatus(http.StatusNotFound), "4xx should be yellow")
	assert.Equal(t, red, colorForStatus(2), "other things should be red")
}

func TestResetColor(t *testing.T) {
	p := LogFormatterParams{}
	assert.Equal(t, string([]byte{27, 91, 48, 109}), p.ResetColor())
}

func TestIsOutputColor(t *testing.T) {
	// test with isTerm flag true.
	p := LogFormatterParams{
		isTerm: true,
	}

	consoleColorMode = autoColor
	assert.Equal(t, true, p.IsOutputColor())

	ForceConsoleColor()
	assert.Equal(t, true, p.IsOutputColor())

	DisableConsoleColor()
	assert.Equal(t, false, p.IsOutputColor())

	// test with isTerm flag false.
	p = LogFormatterParams{
		isTerm: false,
	}

	consoleColorMode = autoColor
	assert.Equal(t, false, p.IsOutputColor())

	ForceConsoleColor()
	assert.Equal(t, true, p.IsOutputColor())

	DisableConsoleColor()
	assert.Equal(t, false, p.IsOutputColor())

	// reset console color mode.
	consoleColorMode = autoColor
}

func TestLoggerWithWriterSkippingPaths(t *testing.T) {
	buffer := new(bytes.Buffer)
	router := New()
	router.Use(LoggerWithWriter(buffer, "/skipped"))
	router.Handle("/logged", func(c *Context) {})
	router.Handle("/skipped", func(c *Context) {})

	performRequest("/logged", router)
	assert.Contains(t, buffer.String(), "200")

	buffer.Reset()
	performRequest("/skipped", router)
	assert.Equal(t, "", buffer.String())
}

func TestLoggerWithConfigSkippingPaths(t *testing.T) {
	buffer := new(bytes.Buffer)
	router := New()
	router.Use(LoggerWithConfig(LoggerConfig{
		Output:    buffer,
		SkipPaths: []string{"/skipped"},
	}))
	router.Handle("/logged", func(c *Context) {})
	router.Handle("/skipped", func(c *Context) {})

	performRequest("/logged", router)
	assert.Contains(t, buffer.String(), "200")

	buffer.Reset()
	performRequest("/skipped", router)
	assert.Equal(t, "", buffer.String())
}

func TestDisableConsoleColor(t *testing.T) {
	New()
	assert.Equal(t, autoColor, consoleColorMode)
	DisableConsoleColor()
	assert.Equal(t, disableColor, consoleColorMode)

	// reset console color mode.
	consoleColorMode = autoColor
}

func TestForceConsoleColor(t *testing.T) {
	New()
	assert.Equal(t, autoColor, consoleColorMode)
	ForceConsoleColor()
	assert.Equal(t, forceColor, consoleColorMode)

	// reset console color mode.
	consoleColorMode = autoColor
}

package gotham

import (
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO
// func debugRoute(httpMethod, absolutePath string, handlers HandlersChain) {
// fu nc debugPrint(format string, values ...interface{}) {

func TestIsDebugging(t *testing.T) {
	SetMode(DebugMode)
	assert.True(t, IsDebugging())
	SetMode(ReleaseMode)
	assert.False(t, IsDebugging())
	SetMode(TestMode)
	assert.False(t, IsDebugging())
}

func TestDebugPrint(t *testing.T) {
	re := captureOutput(t, func() {
		SetMode(DebugMode)
		SetMode(ReleaseMode)
		debugPrint("DEBUG this!")
		SetMode(TestMode)
		debugPrint("DEBUG this!")
		SetMode(DebugMode)
		debugPrint("these are %d %s", 2, "error messages")
		SetMode(TestMode)
	})
	assert.Equal(t, "[GOTHAM-debug] these are 2 error messages\n", re)
}

func TestDebugPrintError(t *testing.T) {
	re := captureOutput(t, func() {
		SetMode(DebugMode)
		debugPrintError(nil)
		debugPrintError(errors.New("this is an error"))
		SetMode(TestMode)
	})
	assert.Equal(t, "[GOTHAM-debug] [ERROR] this is an error\n", re)
}

func TestDebugPrintRoutes(t *testing.T) {
	re := captureOutput(t, func() {
		SetMode(DebugMode)
		debugPrintRoute("/path/to/route/:param", HandlersChain{func(c *Context) {}, handlerNameTest})
		SetMode(TestMode)
	})
	assert.Regexp(t, `^\[GOTHAM-debug\] /path/to/route/:param     --> (.*/vendor/)?github.com/sleep2death/gotham.handlerNameTest \(2 handlers\)\n$`, re)
}

func TestDebugPrintWARNINGDefault(t *testing.T) {
	re := captureOutput(t, func() {
		SetMode(DebugMode)
		debugPrintWARNINGDefault()
		SetMode(TestMode)
	})
	m, e := getMinVer(runtime.Version())
	if e == nil && m <= gSupportMinGoVer {
		assert.Equal(t, "[GOTHAM-debug] [WARNING] Now Gotham requires Go 1.11 or later and Go 1.12 will be required soon.\n[GOTHAM-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.\n", re)
	} else {
		assert.Equal(t, "[GOTHAM-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.\n", re)
	}
}

func TestDebugPrintWARNINGNew(t *testing.T) {
	re := captureOutput(t, func() {
		SetMode(DebugMode)
		debugPrintWARNINGNew()
		SetMode(TestMode)
	})
	assert.Equal(t, "[GOTHAM-debug] [WARNING] Running in \"debug\" mode. Switch to \"release\" mode in production.\n- using env:\texport GOTHAM_MODE=release\n- using code:\tgotham.SetMode(gotham.ReleaseMode)\n", re)
}

func captureOutput(t *testing.T, f func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	defaultWriter := DefaultWriter
	defaultErrorWriter := DefaultErrorWriter
	defer func() {
		DefaultWriter = defaultWriter
		DefaultErrorWriter = defaultErrorWriter
		log.SetOutput(os.Stderr)
	}()
	DefaultWriter = writer
	DefaultErrorWriter = writer
	log.SetOutput(writer)
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		_, err := io.Copy(&buf, reader)
		assert.NoError(t, err)
		out <- buf.String()
	}()
	wg.Wait()
	f()
	writer.Close()
	return <-out
}

func TestGetMinVer(t *testing.T) {
	var m uint64
	var e error
	_, e = getMinVer("go1")
	assert.NotNil(t, e)
	m, e = getMinVer("go1.1")
	assert.Equal(t, uint64(1), m)
	assert.Nil(t, e)
	m, e = getMinVer("go1.1.1")
	assert.Nil(t, e)
	assert.Equal(t, uint64(1), m)
	_, e = getMinVer("go1.1.1.1")
	assert.NotNil(t, e)
}

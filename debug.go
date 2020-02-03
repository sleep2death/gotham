package gotham

import (
	"fmt"
	"html/template"
	"runtime"
	"strconv"
	"strings"
)

const gSupportMinGoVer = 10

var (
	gMode int = debugCode
)

// IsDebugging returns true if the framework is running in debug mode.
// Use SetMode(gotham.ReleaseMode) to disable debug mode.
func IsDebugging() bool {
	return gMode == debugCode
}

// DebugPrintRouteFunc indicates debug log output format.
var DebugPrintRouteFunc func(absolutePath, handlerName string, nuHandlers int)

func debugPrintRoute(absolutePath string, handlers HandlersChain) {
	if IsDebugging() {
		nuHandlers := len(handlers)
		handlerName := nameOfFunction(handlers.Last())
		if DebugPrintRouteFunc == nil {
			debugPrint("%-25s --> %s (%d handlers)\n", absolutePath, handlerName, nuHandlers)
		} else {
			DebugPrintRouteFunc(absolutePath, handlerName, nuHandlers)
		}
	}
}

func debugPrintLoadTemplate(tmpl *template.Template) {
	if IsDebugging() {
		var buf strings.Builder
		for _, tmpl := range tmpl.Templates() {
			buf.WriteString("\t- ")
			buf.WriteString(tmpl.Name())
			buf.WriteString("\n")
		}
		debugPrint("Loaded HTML Templates (%d): \n%s\n", len(tmpl.Templates()), buf.String())
	}
}

func debugPrint(format string, values ...interface{}) {
	if IsDebugging() {
		if !strings.HasSuffix(format, "\n") {
			format += "\n"
		}
		fmt.Fprintf(DefaultWriter, "[GOTHAM-debug] "+format, values...)
	}
}

func getMinVer(v string) (uint64, error) {
	first := strings.IndexByte(v, '.')
	last := strings.LastIndexByte(v, '.')
	if first == last {
		return strconv.ParseUint(v[first+1:], 10, 64)
	}
	return strconv.ParseUint(v[first+1:last], 10, 64)
}

func debugPrintWARNINGDefault() {
	if v, e := getMinVer(runtime.Version()); e == nil && v <= gSupportMinGoVer {
		debugPrint(`[WARNING] Now Gotham requires Go 1.11 or later and Go 1.12 will be required soon.
`)
	}
	debugPrint(`[WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.
`)
}

func debugPrintWARNINGNew() {
	debugPrint(`[WARNING] Running in "debug" mode. Switch to "release" mode in production.
- using env:	export GOTHAM_MODE=release
 - using code:	gotham.SetMode(gotham.ReleaseMode)
`)
}

func debugPrintWARNINGSetHTMLTemplate() {
	debugPrint(`[WARNING] Since SetHTMLTemplate() is NOT thread-safe. It should only be called
at initialization. ie. before any route is registered or the router is listening in a socket:
	router := gotham.Default()
	router.SetHTMLTemplate(template) // << good place
`)
}

func debugPrintError(err error) {
	if err != nil {
		if IsDebugging() {
			fmt.Fprintf(DefaultErrorWriter, "[GOTHAM-debug] [ERROR] %v\n", err)
		}
	}
}

const (
	// DebugMode indicates gotham mode is debug.
	DebugMode = "debug"
	// ReleaseMode indicates gotham mode is release.
	ReleaseMode = "release"
	// TestMode indicates gotham mode is test.
	TestMode = "test"
)
const (
	debugCode = iota
	releaseCode
	testCode
)

// SetMode sets gotham mode according to input string.
func SetMode(value string) {
	switch value {
	case DebugMode, "":
		gMode = debugCode
	case ReleaseMode:
		gMode = releaseCode
	case TestMode:
		gMode = testCode
	default:
		panic("gmode unknown: " + value)
	}
}

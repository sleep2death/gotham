package gotham

import (
	"bufio"
	"math"
	"sync"

	"github.com/golang/protobuf/ptypes/any"
)

const defaultMultipartMemory = 32 << 20 // 32 MB

const abortIndex int8 = math.MaxInt8 / 2

// HandlerFunc defines the handler used by gamerouter middleware as return value.
type HandlerFunc func(*Context)

// HandlersChain defines a HandlerFunc array.
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main one.
func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

// RouteInfo represents a request route's specification which contains path and its handler.
type RouteInfo struct {
	Path        string
	Handler     string
	HandlerFunc HandlerFunc
}

// RoutesInfo defines a RouteInfo array.
type RoutesInfo []RouteInfo

// Router of the gamerouter
type Router struct {
	RouterGroup
	MaxMultipartMemory int64

	// RemoveExtraSlash a parameter can be parsed from the URL even with extra slashes.
	// See the PR #1817 and issue #1644
	RemoveExtraSlash bool

	allNoRoute HandlersChain
	noRoute    HandlersChain
	pool       sync.Pool
	root       *node
}

var _ IRouter = &Router{}

// New returns a new blank Engine instance without any middleware attached.
func New() *Router {
	engine := &Router{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		RemoveExtraSlash:   true,
		MaxMultipartMemory: defaultMultipartMemory,
		root:               new(node),
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		return engine.allocateContext()
	}
	return engine
}

func (engine *Router) allocateContext() *Context {
	return &Context{engine: engine}
}

// NoRoute adds handlers for NoRoute. It return a 404 code by default.
func (engine *Router) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// Use attaches a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (engine *Router) Use(middleware ...HandlerFunc) IRoutes {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	return engine
}

func (engine *Router) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Router) addRoute(path string, handlers HandlersChain) {
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(len(handlers) > 0, "there must be at least one handler")

	// debugPrintRoute(method, path, handlers)
	engine.root.addRoute(path, handlers)
}

func iterate(path, method string, routes RoutesInfo, root *node) RoutesInfo {
	path += root.path
	if len(root.handlers) > 0 {
		handlerFunc := root.handlers.Last()
		routes = append(routes, RouteInfo{
			Path:        path,
			Handler:     nameOfFunction(handlerFunc),
			HandlerFunc: handlerFunc,
		})
	}
	for _, child := range root.children {
		routes = iterate(path, method, routes, child)
	}
	return routes
}

// Serve conforms to the Handler interface.
func (r *Router) Serve(w *bufio.Writer, msg *any.Any) {
	// get context from pool
	c := r.pool.Get().(*Context)
	c.reset()
	rPath := msg.GetTypeUrl()

	// Find route in the tree
	value := r.root.getValue(rPath, nil, false)
	if value.handlers != nil {
		c.handlers = value.handlers
		c.fullPath = value.fullPath
	} else {
		c.handlers = r.allNoRoute
	}

	c.Next()

	// put context back to the pool
	r.pool.Put(c)
}

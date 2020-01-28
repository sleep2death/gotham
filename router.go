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
	trees      methodTrees
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
		trees:              make(methodTrees, 0, 9),
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
	root := engine.trees.get("any")
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		engine.trees = append(engine.trees, methodTree{method: "any", root: root})
	}
	root.addRoute(path, handlers)
}

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (engine *Router) Routes() (routes RoutesInfo) {
	for _, tree := range engine.trees {
		routes = iterate("", tree.method, routes, tree.root)
	}
	return routes
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
func (engine *Router) ServeProto(w *bufio.Writer, msg *any.Any) {
	// get context from pool
	c := engine.pool.Get().(*Context)
	c.reset()
	rPath := msg.GetTypeUrl()

	// root of the tree
	root := engine.trees[0].root

	// Find route in the tree
	value := root.getValue(rPath, nil, false)
	if value.handlers != nil {
		c.handlers = value.handlers
		c.fullPath = value.fullPath
	} else {
		c.handlers = engine.allNoRoute
	}

	c.Next()

	// put context back to the pool
	engine.pool.Put(c)
}

// Serve conforms to the Handler interface.
func (engine *Router) Serve(rep *Response, req *Request) {
	c := engine.pool.Get().(*Context)
	c.reset()

	c.Response = rep
	c.Request = req
	engine.handleRequest(c)

	engine.pool.Put(c)
}

func (engine *Router) handleRequest(c *Context) {
	method := "any"
	rPath := c.Request.Path
	unescape := false

	if engine.RemoveExtraSlash {
		rPath = cleanPath(rPath)
	}

	// Find root of the tree for the given HTTP method
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != method {
			continue
		}
		root := t[i].root
		// Find route in tree
		value := root.getValue(rPath, c.Request.Params, unescape)
		if value.handlers != nil {
			c.handlers = value.handlers
			c.Request.Params = value.params
			c.fullPath = value.fullPath
			c.Next()
			// c.writermem.WriteHeaderNow()
			return
		}
		break
	}

	c.handlers = engine.allNoRoute
	c.Next()
}

package gotham

import (
	"bufio"
	"math"
	"sync"
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

// New returns a new blank Router instance without any middleware attached.
func New() *Router {
	router := &Router{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		RemoveExtraSlash:   true,
		MaxMultipartMemory: defaultMultipartMemory,
		root:               new(node),
	}
	router.RouterGroup.engine = router
	router.pool.New = func() interface{} {
		return router.allocateContext()
	}
	return router
}

func (router *Router) allocateContext() *Context {
	return &Context{router: router}
}

// NoRoute adds handlers for NoRoute. It return a 404 code by default.
func (router *Router) NoRoute(handlers ...HandlerFunc) {
	router.noRoute = handlers
	router.rebuild404Handlers()
}

// Use attaches a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (router *Router) Use(middleware ...HandlerFunc) IRoutes {
	router.RouterGroup.Use(middleware...)
	router.rebuild404Handlers()
	return router
}

func (router *Router) rebuild404Handlers() {
	router.allNoRoute = router.combineHandlers(router.noRoute)
}

func (router *Router) addRoute(path string, handlers HandlersChain) {
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(len(handlers) > 0, "there must be at least one handler")

	// debugPrintRoute(method, path, handlers)
	router.root.addRoute(path, handlers)
}

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (router *Router) Routes() (routes RoutesInfo) {
	tree := router.root
	routes = iterate("", routes, tree)
	return routes
}

func iterate(path string, routes RoutesInfo, root *node) RoutesInfo {
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
		routes = iterate(path, routes, child)
	}
	return routes
}

func (r *Router) Run(addr string) (err error) {
	err = ListenAndServe(addr, r)
	return
}

// Serve conforms to the Handler interface.
func (r *Router) ServeProto(w *bufio.Writer, req *Request) {
	// get context from pool
	c := r.pool.Get().(*Context)
	c.reset()

	c.writer = w
	c.request = req
	// Find route in the tree
	value := r.root.getValue(c.request.url, nil, false)
	if value.handlers != nil {
		c.handlers = value.handlers
		c.fullPath = value.fullPath
	} else {
		// no route was found
		c.handlers = r.allNoRoute
	}
	c.Next()

	// put context back to the pool
	r.pool.Put(c)
}

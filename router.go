package gotham

import (
	"math"
	"net/http"
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

	allNoRoute HandlersChain
	noRoute    HandlersChain
	pool       sync.Pool

	nodes  pnodes
	groups []*RouterGroup
}

// var _ IRouter = &Router{}

// New returns a new blank Router instance without any middleware attached.
func New() *Router {
	debugPrintWARNINGNew()
	router := &Router{
		RouterGroup: RouterGroup{
			handlers: nil,
			name:     "default",
		},
		nodes: make(pnodes, 0),
	}
	router.RouterGroup.router = router
	router.pool.New = func() interface{} {
		return router.allocateContext()
	}
	return router
}

// Default returns a Router instance with the Logger and Recovery middleware already attached.
func Default() *Router {
	debugPrintWARNINGDefault()
	router := New()
	router.Use(Logger(), Recovery())
	router.NoRoute(DefaultNoRouteHandler)
	return router
}

func DefaultNoRouteHandler(c *Context) {
	c.writeError(http.StatusNotFound, "route not found")
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

func (router *Router) Group(name string) *RouterGroup {
	for _, g := range router.groups {
		if g.name == name {
			return g
		}
	}
	g := &RouterGroup{name: name, router: router}
	return g
}

func (router *Router) rebuild404Handlers() {
	finalSize := len(router.handlers) + len(router.noRoute)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	router.allNoRoute = append(router.handlers, router.noRoute...)
}

func (router *Router) addRoute(path string, handlers HandlersChain) {
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(len(handlers) > 0, "there must be at least one handler")

	debugPrintRoute(path, handlers)
}

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (router *Router) Routes() (routes RoutesInfo) {
	routes = iterate(routes, router.nodes)
	return routes
}

func iterate(routes RoutesInfo, nodes pnodes) RoutesInfo {
	for _, node := range nodes {
		handlerFunc := node.handlers.Last()
		routes = append(routes, RouteInfo{
			Path:        node.name,
			Handler:     nameOfFunction(handlerFunc),
			HandlerFunc: handlerFunc,
		})
	}
	return routes
}

func (r *Router) Run(addr string, codec Codec) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()
	err = ListenAndServe(addr, r, codec)
	return
}

// HandleContext re-enter a context that has been rewritten.
// This can be done by setting c.Request.URL.Path to your new target.
// Disclaimer: You can loop yourself to death with this, use wisely.
func (router *Router) HandleContext(c *Context) {
	oldIndexValue := c.index
	c.reset()
	router.handleProtoRequest(c)
	c.index = oldIndexValue
}

// Serve conforms to the Handler interface.
func (r *Router) ServeProto(w ResponseWriter, req *Request) {
	// get context from pool
	c := r.pool.Get().(*Context)
	// reset context
	c.Writer = w
	c.Request = req
	c.reset()

	r.handleProtoRequest(c)

	// put context back to the pool
	r.pool.Put(c)
}

func (router *Router) handleProtoRequest(c *Context) {
	// Find route in the tree
	// url, _ := fixPath(c.Request.URL)
	value := router.nodes.get(c.Request.TypeURL)
	if value != nil {
		c.handlers = value.handlers
	} else {
		// no route was found
		c.handlers = router.allNoRoute
	}
	c.Next()
}

package gotham

// IHandlers defines all routers which have handlers,
// and a unique name .
type IHandlers interface {
	Handlers() HandlersChain
	Name() string
}

// IRoutes defines all router handle interface.
type IRoutes interface {
	Handle(string, ...HandlerFunc) IRoutes
	Use(...HandlerFunc) IRoutes
}

// RouterGroup is used internally to configure router, a RouterGroup is associated with
// a prefix and an array of handlers (middleware).
type RouterGroup struct {
	handlers HandlersChain
	name     string
	router   *Router
}

// Use adds middleware to the group, see example code in GitHub.
func (group *RouterGroup) Use(middlewares ...HandlerFunc) IRoutes {
	group.handlers = append(group.handlers, middlewares...)
	return group
}

// Handle
func (group *RouterGroup) Handle(name string, handlers ...HandlerFunc) IRoutes {
	var pn *pnode = group.router.nodes.get(name)
	if pn == nil {
		pn = &pnode{name: name}
		group.router.nodes = append(group.router.nodes, pn)
	}

	pn.addGroup(group.router)
	pn.addGroup(group)
	pn.rebuildHandlers(handlers...)

	return group
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware that can and should be shared among different routes.
// See the example code in GitHub.
func (group *RouterGroup) Handlers() HandlersChain {
	return group.handlers
}

func (group *RouterGroup) Name() string {
	return group.name
}

type pnodes []*pnode

type pnode struct {
	name      string
	groups    []IHandlers
	phandlers HandlersChain
	handlers  HandlersChain
}

func (pn *pnode) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(pn.handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, pn.handlers)
	copy(mergedHandlers[len(pn.handlers):], handlers)
	return mergedHandlers
}

func (pn *pnode) addGroup(group IHandlers) {
	for _, g := range pn.groups {
		if g.Name() == group.Name() {
			return
		}
	}
	pn.groups = append(pn.groups, group)
}

func (pn *pnode) rebuildHandlers(handlers ...HandlerFunc) {
	pn.phandlers = append(pn.phandlers, handlers...)
	pn.handlers = pn.handlers[:0]
	for _, group := range pn.groups {
		pn.handlers = pn.combineHandlers(group.Handlers())
	}
	pn.handlers = pn.combineHandlers(pn.phandlers)
}

func (nodes pnodes) get(name string) *pnode {
	for _, n := range nodes {
		if n.name == name {
			return n
		}
	}
	return nil
}

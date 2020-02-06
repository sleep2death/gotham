package gotham

type IHandlers interface {
	Handlers() HandlersChain
	Name() string
}

type IRoutes interface {
	Handle(string, ...HandlerFunc) IRoutes
	Use(...HandlerFunc) IRoutes
}

type RouterGroup struct {
	handlers HandlersChain
	name     string
	router   *Router
}

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

func (group *RouterGroup) Handlers() HandlersChain {
	return group.handlers
}

func (group *RouterGroup) Name() string {
	return group.name
}

func (group *RouterGroup) Use(middlewares ...HandlerFunc) IRoutes {
	group.handlers = append(group.handlers, middlewares...)
	return group
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

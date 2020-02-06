package gotham

type PRouter struct {
	PGroup
	groups []*PGroup
	nodes  pnodes
}

func NewRouter() *PRouter {
	return &PRouter{
		PGroup: PGroup{
			name: "default",
			root: true,
		},
	}
}

func (router *PRouter) Group(name string) *PGroup {
	for _, group := range router.groups {
		if group.name == name {
			return group
		}
	}
	group := &PGroup{name: name, router: router}
	router.groups = append(router.groups, group)
	return group
}

func (router *PRouter) Handle(name string, handlers ...HandlerFunc) {
	var pn *pnode = router.nodes.get(name)
	if pn == nil {
		pn = &pnode{name: name}
		router.nodes = append(router.nodes, pn)
	}
	pn.addGroup(router)
	pn.rebuildHandlers(handlers...)
}

type IHandlers interface {
	Handlers() HandlersChain
	Router() *PRouter
	Name() string
}

type PGroup struct {
	handlers HandlersChain
	name     string
	root     bool
	router   *PRouter
}

func (group *PGroup) Handle(name string, handlers ...HandlerFunc) {
	var pn *pnode = group.router.nodes.get(name)
	if pn == nil {
		pn = &pnode{name: name}
		group.router.nodes = append(group.router.nodes, pn)
	}

	pn.addGroup(group.router)
	pn.addGroup(group)
	pn.rebuildHandlers(handlers...)
}

func (group *PGroup) Handlers() HandlersChain {
	return group.handlers
}

func (group *PGroup) Router() *PRouter {
	return group.router
}

func (group *PGroup) Name() string {
	return group.name
}

func (group *PGroup) Use(middlewares ...HandlerFunc) {
	group.handlers = append(group.handlers, middlewares...)
}

type pnodes []*pnode

type pnode struct {
	name     string
	groups   []IHandlers
	handlers HandlersChain
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
	pn.handlers = pn.handlers[:0]
	for _, group := range pn.groups {
		pn.handlers = pn.combineHandlers(group.Handlers())
	}
	pn.handlers = pn.combineHandlers(handlers)
}

func (nodes pnodes) get(name string) *pnode {
	for _, n := range nodes {
		if n.name == name {
			return n
		}
	}
	return nil
}

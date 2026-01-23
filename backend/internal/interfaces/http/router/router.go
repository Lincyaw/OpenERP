package router

import (
	"github.com/gin-gonic/gin"
)

// RouteRegistrar defines the interface for registering routes
type RouteRegistrar interface {
	RegisterRoutes(rg *gin.RouterGroup)
}

// Router manages HTTP route registration
type Router struct {
	engine     *gin.Engine
	apiVersion string
	registrars []RouteRegistrar
}

// RouterOption is a functional option for Router configuration
type RouterOption func(*Router)

// WithAPIVersion sets the API version prefix (e.g., "v1", "v2")
func WithAPIVersion(version string) RouterOption {
	return func(r *Router) {
		r.apiVersion = version
	}
}

// NewRouter creates a new Router instance
func NewRouter(engine *gin.Engine, opts ...RouterOption) *Router {
	r := &Router{
		engine:     engine,
		apiVersion: "v1",
		registrars: make([]RouteRegistrar, 0),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Register adds a RouteRegistrar to be registered later
func (r *Router) Register(registrar RouteRegistrar) *Router {
	r.registrars = append(r.registrars, registrar)
	return r
}

// Setup registers all routes with the engine
func (r *Router) Setup() {
	// Create versioned API group
	api := r.engine.Group("/api/" + r.apiVersion)

	// Register all route registrars
	for _, registrar := range r.registrars {
		registrar.RegisterRoutes(api)
	}
}

// DomainGroup creates a route group for a specific domain
type DomainGroup struct {
	name       string
	prefix     string
	handlers   []gin.HandlerFunc
	routes     []routeDefinition
	subgroups  []*DomainGroup
	middleware []gin.HandlerFunc
}

type routeDefinition struct {
	method      string
	path        string
	handlers    []gin.HandlerFunc
	description string
}

// NewDomainGroup creates a new domain-specific route group
func NewDomainGroup(name, prefix string) *DomainGroup {
	return &DomainGroup{
		name:       name,
		prefix:     prefix,
		handlers:   make([]gin.HandlerFunc, 0),
		routes:     make([]routeDefinition, 0),
		subgroups:  make([]*DomainGroup, 0),
		middleware: make([]gin.HandlerFunc, 0),
	}
}

// Use adds middleware to this group
func (dg *DomainGroup) Use(middleware ...gin.HandlerFunc) *DomainGroup {
	dg.middleware = append(dg.middleware, middleware...)
	return dg
}

// GET registers a GET route
func (dg *DomainGroup) GET(path string, handlers ...gin.HandlerFunc) *DomainGroup {
	dg.routes = append(dg.routes, routeDefinition{
		method:   "GET",
		path:     path,
		handlers: handlers,
	})
	return dg
}

// POST registers a POST route
func (dg *DomainGroup) POST(path string, handlers ...gin.HandlerFunc) *DomainGroup {
	dg.routes = append(dg.routes, routeDefinition{
		method:   "POST",
		path:     path,
		handlers: handlers,
	})
	return dg
}

// PUT registers a PUT route
func (dg *DomainGroup) PUT(path string, handlers ...gin.HandlerFunc) *DomainGroup {
	dg.routes = append(dg.routes, routeDefinition{
		method:   "PUT",
		path:     path,
		handlers: handlers,
	})
	return dg
}

// PATCH registers a PATCH route
func (dg *DomainGroup) PATCH(path string, handlers ...gin.HandlerFunc) *DomainGroup {
	dg.routes = append(dg.routes, routeDefinition{
		method:   "PATCH",
		path:     path,
		handlers: handlers,
	})
	return dg
}

// DELETE registers a DELETE route
func (dg *DomainGroup) DELETE(path string, handlers ...gin.HandlerFunc) *DomainGroup {
	dg.routes = append(dg.routes, routeDefinition{
		method:   "DELETE",
		path:     path,
		handlers: handlers,
	})
	return dg
}

// Group creates a sub-group within this domain
func (dg *DomainGroup) Group(name, prefix string) *DomainGroup {
	subgroup := NewDomainGroup(name, prefix)
	dg.subgroups = append(dg.subgroups, subgroup)
	return subgroup
}

// RegisterRoutes implements RouteRegistrar interface
func (dg *DomainGroup) RegisterRoutes(rg *gin.RouterGroup) {
	// Create group with prefix
	group := rg.Group(dg.prefix)

	// Apply middleware
	if len(dg.middleware) > 0 {
		group.Use(dg.middleware...)
	}

	// Register routes
	for _, route := range dg.routes {
		switch route.method {
		case "GET":
			group.GET(route.path, route.handlers...)
		case "POST":
			group.POST(route.path, route.handlers...)
		case "PUT":
			group.PUT(route.path, route.handlers...)
		case "PATCH":
			group.PATCH(route.path, route.handlers...)
		case "DELETE":
			group.DELETE(route.path, route.handlers...)
		}
	}

	// Register subgroups recursively
	for _, subgroup := range dg.subgroups {
		subgroup.RegisterRoutes(group)
	}
}

// Name returns the group name
func (dg *DomainGroup) Name() string {
	return dg.name
}

// Prefix returns the group prefix
func (dg *DomainGroup) Prefix() string {
	return dg.prefix
}

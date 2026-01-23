package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewRouter(t *testing.T) {
	engine := gin.New()
	r := NewRouter(engine)

	assert.NotNil(t, r)
	assert.Equal(t, "v1", r.apiVersion)
	assert.Empty(t, r.registrars)
}

func TestRouterWithAPIVersion(t *testing.T) {
	engine := gin.New()
	r := NewRouter(engine, WithAPIVersion("v2"))

	assert.Equal(t, "v2", r.apiVersion)
}

func TestRouterRegister(t *testing.T) {
	engine := gin.New()
	r := NewRouter(engine)

	group := NewDomainGroup("test", "/test")
	r.Register(group)

	assert.Len(t, r.registrars, 1)
}

func TestRouterSetup(t *testing.T) {
	engine := gin.New()
	r := NewRouter(engine, WithAPIVersion("v1"))

	group := NewDomainGroup("test", "/test")
	group.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.Register(group)
	r.Setup()

	// Test the route was registered
	req := httptest.NewRequest("GET", "/api/v1/test/ping", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

func TestDomainGroup(t *testing.T) {
	t.Run("creates group with name and prefix", func(t *testing.T) {
		g := NewDomainGroup("catalog", "/catalog")
		assert.Equal(t, "catalog", g.Name())
		assert.Equal(t, "/catalog", g.Prefix())
	})

	t.Run("registers GET route", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("test", "/test")
		g.GET("/items", func(c *gin.Context) {
			c.String(http.StatusOK, "items")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		req := httptest.NewRequest("GET", "/api/v1/test/items", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("registers POST route", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("test", "/test")
		g.POST("/items", func(c *gin.Context) {
			c.String(http.StatusCreated, "created")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		req := httptest.NewRequest("POST", "/api/v1/test/items", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("registers PUT route", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("test", "/test")
		g.PUT("/items/:id", func(c *gin.Context) {
			c.String(http.StatusOK, "updated")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		req := httptest.NewRequest("PUT", "/api/v1/test/items/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("registers PATCH route", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("test", "/test")
		g.PATCH("/items/:id", func(c *gin.Context) {
			c.String(http.StatusOK, "patched")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		req := httptest.NewRequest("PATCH", "/api/v1/test/items/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("registers DELETE route", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("test", "/test")
		g.DELETE("/items/:id", func(c *gin.Context) {
			c.String(http.StatusNoContent, "")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		req := httptest.NewRequest("DELETE", "/api/v1/test/items/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("applies middleware", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("test", "/test")

		// Add middleware that sets a header
		g.Use(func(c *gin.Context) {
			c.Header("X-Test-Middleware", "applied")
			c.Next()
		})

		g.GET("/items", func(c *gin.Context) {
			c.String(http.StatusOK, "ok")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		req := httptest.NewRequest("GET", "/api/v1/test/items", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, "applied", w.Header().Get("X-Test-Middleware"))
	})

	t.Run("creates subgroups", func(t *testing.T) {
		engine := gin.New()
		g := NewDomainGroup("catalog", "/catalog")

		products := g.Group("products", "/products")
		products.GET("", func(c *gin.Context) {
			c.String(http.StatusOK, "products list")
		})

		categories := g.Group("categories", "/categories")
		categories.GET("", func(c *gin.Context) {
			c.String(http.StatusOK, "categories list")
		})

		api := engine.Group("/api/v1")
		g.RegisterRoutes(api)

		// Test products route
		req1 := httptest.NewRequest("GET", "/api/v1/catalog/products", nil)
		w1 := httptest.NewRecorder()
		engine.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		assert.Equal(t, "products list", w1.Body.String())

		// Test categories route
		req2 := httptest.NewRequest("GET", "/api/v1/catalog/categories", nil)
		w2 := httptest.NewRecorder()
		engine.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
		assert.Equal(t, "categories list", w2.Body.String())
	})
}

func TestMultipleDomainGroups(t *testing.T) {
	engine := gin.New()
	r := NewRouter(engine)

	catalog := NewDomainGroup("catalog", "/catalog")
	catalog.GET("/products", func(c *gin.Context) {
		c.String(http.StatusOK, "products")
	})

	partner := NewDomainGroup("partner", "/partner")
	partner.GET("/customers", func(c *gin.Context) {
		c.String(http.StatusOK, "customers")
	})

	r.Register(catalog).Register(partner)
	r.Setup()

	// Test catalog route
	req1 := httptest.NewRequest("GET", "/api/v1/catalog/products", nil)
	w1 := httptest.NewRecorder()
	engine.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "products", w1.Body.String())

	// Test partner route
	req2 := httptest.NewRequest("GET", "/api/v1/partner/customers", nil)
	w2 := httptest.NewRecorder()
	engine.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "customers", w2.Body.String())
}

func TestChainedMethodCalls(t *testing.T) {
	engine := gin.New()
	r := NewRouter(engine)

	g := NewDomainGroup("test", "/test")
	g.GET("/a", func(c *gin.Context) { c.String(http.StatusOK, "a") }).
		POST("/b", func(c *gin.Context) { c.String(http.StatusOK, "b") }).
		PUT("/c", func(c *gin.Context) { c.String(http.StatusOK, "c") })

	r.Register(g).Setup()

	// All routes should be registered
	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/test/a"},
		{"POST", "/api/v1/test/b"},
		{"PUT", "/api/v1/test/c"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Route %s %s should work", tt.method, tt.path)
	}
}

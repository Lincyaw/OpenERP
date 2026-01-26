package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSwaggerProtection_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := SwaggerConfig{
		Enabled:     false,
		RequireAuth: false,
		AllowedIPs:  []string{},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "not_found")
}

func TestSwaggerProtection_Enabled_NoRestrictions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: false,
		AllowedIPs:  []string{},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSwaggerProtection_IPWhitelist_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: false,
		AllowedIPs:  []string{"127.0.0.1"},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSwaggerProtection_IPWhitelist_Denied(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: false,
		AllowedIPs:  []string{"10.0.0.1"},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "forbidden")
}

func TestSwaggerProtection_CIDRWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: false,
		AllowedIPs:  []string{"10.0.0.0/8"}, // Allow 10.x.x.x range
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, nil), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	// Test allowed IP in CIDR range
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	req.RemoteAddr = "10.50.100.200:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test denied IP outside CIDR range
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/swagger/index.html", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSwaggerProtection_RequireAuth_WithMockJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock JWT middleware that always denies
	mockJWTDeny := func(c *gin.Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: true,
		AllowedIPs:  []string{},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, mockJWTDeny), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSwaggerProtection_RequireAuth_WithMockJWT_Allow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock JWT middleware that always allows
	mockJWTAllow := func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	}

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: true,
		AllowedIPs:  []string{},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, mockJWTAllow), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSwaggerProtection_CombinedProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock JWT middleware that always allows
	mockJWTAllow := func(c *gin.Context) {
		c.Set("user_id", "test-user")
		c.Next()
	}

	cfg := SwaggerConfig{
		Enabled:     true,
		RequireAuth: true,
		AllowedIPs:  []string{"127.0.0.1"},
	}

	router := gin.New()
	router.GET("/swagger/*any", SwaggerProtection(cfg, mockJWTAllow), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "swagger"})
	})

	// Test 1: Correct IP + valid auth = allowed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/swagger/index.html", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test 2: Wrong IP = denied (IP check comes first)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/swagger/index.html", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestIsIPAllowed(t *testing.T) {
	tests := []struct {
		name        string
		ip          string
		allowedIPs  []string
		allowedCIDR []string
		want        bool
	}{
		{
			name:       "exact IP match",
			ip:         "192.168.1.1",
			allowedIPs: []string{"192.168.1.1"},
			want:       true,
		},
		{
			name:       "no match",
			ip:         "192.168.1.2",
			allowedIPs: []string{"192.168.1.1"},
			want:       false,
		},
		{
			name:        "CIDR match",
			ip:          "10.0.0.5",
			allowedCIDR: []string{"10.0.0.0/8"},
			want:        true,
		},
		{
			name:        "CIDR no match",
			ip:          "11.0.0.5",
			allowedCIDR: []string{"10.0.0.0/8"},
			want:        false,
		},
		{
			name:       "localhost IPv4",
			ip:         "127.0.0.1",
			allowedIPs: []string{"127.0.0.1"},
			want:       true,
		},
		{
			name:       "IPv6 localhost",
			ip:         "::1",
			allowedIPs: []string{"::1"},
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allowedIPs []net.IP
			for _, ipStr := range tt.allowedIPs {
				ip := net.ParseIP(ipStr)
				if ip != nil {
					allowedIPs = append(allowedIPs, ip)
				}
			}

			var allowedNets []*net.IPNet
			for _, cidr := range tt.allowedCIDR {
				_, network, err := net.ParseCIDR(cidr)
				if err == nil {
					allowedNets = append(allowedNets, network)
				}
			}

			ip := net.ParseIP(tt.ip)
			got := isIPAllowed(ip, allowedIPs, allowedNets)
			assert.Equal(t, tt.want, got)
		})
	}
}

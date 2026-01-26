package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SwaggerConfig holds configuration for Swagger endpoint protection
type SwaggerConfig struct {
	Enabled     bool     // Whether Swagger endpoint is enabled
	RequireAuth bool     // Require JWT authentication to access Swagger
	AllowedIPs  []string // IP whitelist (CIDR notation supported, empty = allow all)
}

// SwaggerProtection returns a middleware that protects Swagger endpoints
// based on the provided configuration.
//
// Protection modes:
// 1. Disabled: Returns 404 for all Swagger requests
// 2. RequireAuth: Requires valid JWT token to access Swagger
// 3. IP Whitelist: Only allows requests from specified IPs/CIDRs
// 4. Combination: Can combine RequireAuth + IP whitelist for maximum security
func SwaggerProtection(cfg SwaggerConfig, jwtMiddleware gin.HandlerFunc) gin.HandlerFunc {
	// Parse CIDR networks on initialization for performance
	var allowedNets []*net.IPNet
	var allowedIPs []net.IP
	if len(cfg.AllowedIPs) > 0 {
		for _, ipStr := range cfg.AllowedIPs {
			if strings.Contains(ipStr, "/") {
				// CIDR notation
				_, network, err := net.ParseCIDR(ipStr)
				if err == nil {
					allowedNets = append(allowedNets, network)
				}
			} else {
				// Single IP
				ip := net.ParseIP(ipStr)
				if ip != nil {
					allowedIPs = append(allowedIPs, ip)
				}
			}
		}
	}

	return func(c *gin.Context) {
		// If Swagger is disabled, return 404
		if !cfg.Enabled {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "API documentation is not available",
			})
			return
		}

		// Check IP whitelist if configured
		if len(cfg.AllowedIPs) > 0 {
			clientIP := getClientIP(c)
			if !isIPAllowed(clientIP, allowedIPs, allowedNets) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":   "forbidden",
					"message": "Access to API documentation is restricted",
				})
				return
			}
		}

		// Check JWT authentication if required
		if cfg.RequireAuth && jwtMiddleware != nil {
			// Store original abort status
			jwtMiddleware(c)
			// If JWT middleware aborted the request, stop processing
			if c.IsAborted() {
				return
			}
		}

		c.Next()
	}
}

// getClientIP extracts the client IP from the request
// It handles X-Forwarded-For and X-Real-IP headers for proxied requests
func getClientIP(c *gin.Context) net.IP {
	// Try Gin's built-in ClientIP (handles trusted proxies)
	clientIP := c.ClientIP()
	if clientIP != "" {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			return ip
		}
	}

	// Fallback to remote address
	remoteAddr := c.Request.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		// RemoteAddr might not have port
		host = remoteAddr
	}
	return net.ParseIP(host)
}

// isIPAllowed checks if the given IP is in the allowed list
func isIPAllowed(ip net.IP, allowedIPs []net.IP, allowedNets []*net.IPNet) bool {
	if ip == nil {
		return false
	}

	// Check exact IP matches
	for _, allowedIP := range allowedIPs {
		if allowedIP.Equal(ip) {
			return true
		}
	}

	// Check CIDR ranges
	for _, network := range allowedNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

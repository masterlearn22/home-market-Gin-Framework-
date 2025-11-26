package middleware

import (
	"net/http"
	"strings"

	"home-market/pkg"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Set ke context Gin (setara Locals di Fiber)
		c.Set("user_id", claims.UserID)
		c.Set("role_id", claims.RoleID)
		c.Set("role_name", claims.RoleName)
		c.Set("permissions", claims.Permissions)

		c.Next()
	}
}

func RoleAllowed(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleAny, exists := c.Get("role_name")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role missing in context"})
			c.Abort()
			return
		}

		role, ok := roleAny.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid role type"})
			c.Abort()
			return
		}

		for _, r := range allowedRoles {
			if strings.EqualFold(role, r) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: role not allowed"})
		c.Abort()
	}
}

func PermissionRequired(needed string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "no permissions found"})
			c.Abort()
			return
		}

		perms, ok := raw.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid permissions format"})
			c.Abort()
			return
		}

		for _, p := range perms {
			if p == needed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error": "permission denied: needed '" + needed + "'",
		})
		c.Abort()
	}
}

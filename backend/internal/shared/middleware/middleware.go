package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"evently/internal/shared/config"
	"evently/internal/shared/utils/response"
	"evently/internal/users"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// creates a JWT authentication middleware
func JWTAuth() gin.HandlerFunc {
	return JWTAuthWithConfig(config.Load())
}

// creates a JWT authentication middleware with config(user payload from token)
func JWTAuthWithConfig(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.RespondJSON(c, "error", http.StatusUnauthorized, "Authorization header is required", nil, nil)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.RespondJSON(c, "error", http.StatusUnauthorized, "authorization header format must be Bearer {token}", nil, nil)
			c.Abort()
			return
		}

		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			response.RespondJSON(c, "error", http.StatusUnauthorized, "invalid or expired token", nil, nil)
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if tokenType, ok := claims["type"]; !ok || tokenType != "access" {
				response.RespondJSON(c, "error", http.StatusUnauthorized, "invalid token type", nil, nil)
				c.Abort()
				return
			}
			log.Println("JWT claims:", claims)
			c.Set("user_id", claims["user_id"])
			c.Set("user_email", claims["email"])
			c.Set("user_role", claims["role"])
		}

		c.Next()
	}
}

// checks if user has required role
func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			response.RespondJSON(c, "error", http.StatusUnauthorized, "user role not found in context", nil, nil)
			c.Abort()
			return
		}

		if userRole.(string) != requiredRole {
			response.RespondJSON(c, "error", http.StatusForbidden, "Insufficient permissions", nil, nil)
			c.Abort()
			return
		}
		fmt.Print("userRole:", userRole)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRole(string(users.RoleAdmin))
}

// checks if user has any of the required roles
func RequireRoles(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			response.RespondJSON(c, "error", http.StatusUnauthorized, "user role not found in context", nil, nil)
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range requiredRoles {
			if userRole.(string) == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			response.RespondJSON(c, "error", http.StatusForbidden, "Insufficient permissions", nil, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}

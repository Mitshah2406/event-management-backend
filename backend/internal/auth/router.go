package auth

import (
	"evently/internal/shared/config"
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

// Router handles auth-related routes
type Router struct {
	controller *Controller
	config     *config.Config
}

// NewRouter creates a new auth router
func NewRouter(controller *Controller) *Router {
	return &Router{
		controller: controller,
		config:     config.Load(), // Load config for middleware
	}
}

// SetupRoutes registers all auth routes
func (authRouter *Router) SetupRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		// Public routes (no authentication required)
		auth.POST("/register", authRouter.controller.Register)
		auth.POST("/login", authRouter.controller.Login)
		auth.POST("/refresh", authRouter.controller.RefreshToken)
		auth.POST("/logout", authRouter.controller.Logout)

		// Protected routes (authentication required)
		protected := auth.Group("")
		protected.Use(middleware.JWTAuthWithConfig(authRouter.config))
		{
			protected.PUT("/change-password", authRouter.controller.ChangePassword)
			protected.GET("/me", authRouter.controller.GetMe)
		}
	}
}

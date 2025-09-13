package auth

import (
	"evently/internal/shared/config"
	"evently/internal/shared/middleware"

	"github.com/gin-gonic/gin"
)

type Router struct {
	controller *Controller
	config     *config.Config
}

func NewRouter(controller *Controller) *Router {
	return &Router{
		controller: controller,
		config:     config.Load(), // Load config for middleware
	}
}

func (authRouter *Router) SetupRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		// Public routes
		auth.POST("/register", authRouter.controller.Register)
		auth.POST("/login", authRouter.controller.Login)
		auth.POST("/refresh", authRouter.controller.RefreshToken)
		auth.POST("/logout", authRouter.controller.Logout)

		// Protected routes
		protected := auth.Group("")
		protected.Use(middleware.JWTAuthWithConfig(authRouter.config))
		{
			protected.PUT("/change-password", authRouter.controller.ChangePassword)
			protected.GET("/me", authRouter.controller.GetMe)
		}
	}
}

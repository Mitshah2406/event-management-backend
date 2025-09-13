package auth

import (
	"evently/internal/shared/utils/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Controller struct {
	service   Service
	validator *validator.Validate
}

func NewController(service Service) *Controller {
	return &Controller{
		service:   service,
		validator: validator.New(),
	}
}

func (c *Controller) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	if err := c.validator.Struct(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Validation failed", nil, err.Error())
		return
	}

	resp, err := c.service.Register(ctx.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrUserAlreadyExists:
			response.RespondJSON(ctx, "error", http.StatusConflict, "User with this email already exists", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to register user", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusCreated, "User registered successfully", resp, nil)
}

func (c *Controller) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	if err := c.validator.Struct(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Validation failed", nil, err.Error())
		return
	}

	resp, err := c.service.Login(ctx.Request.Context(), &req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid email or password", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to login", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Login successful", resp, nil)
}

func (c *Controller) RefreshToken(ctx *gin.Context) {
	var req RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	if err := c.validator.Struct(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Validation failed", nil, err.Error())
		return
	}

	tokenPair, err := c.service.RefreshToken(ctx.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case ErrInvalidToken, ErrTokenExpired:
			response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Invalid or expired refresh token", nil, nil)
		case ErrUserNotFound:
			response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not found", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to refresh token", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Token refreshed successfully", tokenPair, nil)
}

func (c *Controller) Logout(ctx *gin.Context) {
	var req LogoutRequest
	ctx.ShouldBindJSON(&req) // Optional body

	response.RespondJSON(ctx, "success", http.StatusOK, "Logged out successfully", nil, nil)
}

func (c *Controller) ChangePassword(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	var req ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Invalid request body", nil, err.Error())
		return
	}

	if err := c.validator.Struct(&req); err != nil {
		response.RespondJSON(ctx, "error", http.StatusBadRequest, "Validation failed", nil, err.Error())
		return
	}

	err := c.service.ChangePassword(ctx.Request.Context(), userID.(string), &req)
	if err != nil {
		switch err {
		case ErrInvalidCredentials:
			response.RespondJSON(ctx, "error", http.StatusUnauthorized, "Current password is incorrect", nil, nil)
		case ErrUserNotFound:
			response.RespondJSON(ctx, "error", http.StatusNotFound, "User not found", nil, nil)
		default:
			response.RespondJSON(ctx, "error", http.StatusInternalServerError, "Failed to change password", nil, nil)
		}
		return
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "Password changed successfully", nil, nil)
}

func (c *Controller) GetMe(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		response.RespondJSON(ctx, "error", http.StatusUnauthorized, "User not authenticated", nil, nil)
		return
	}

	email, _ := ctx.Get("user_email")
	role, _ := ctx.Get("user_role")

	userData := map[string]interface{}{
		"id":    userID,
		"email": email,
		"role":  role,
	}

	response.RespondJSON(ctx, "success", http.StatusOK, "User data retrieved successfully", userData, nil)
}

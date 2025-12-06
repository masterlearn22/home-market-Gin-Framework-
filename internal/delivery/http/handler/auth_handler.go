package handler

import (
	"net/http"

	entity "home-market/internal/domain"
	service "home-market/internal/service/postgresql"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req struct {
        Username string `json:"username" binding:"required"`
        Email  string `json:"email"  binding:"required,email"`
        FullName string `json:"fullName" binding:"required"`
        Password string `json:"password" binding:"required,min=6"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
        return
    }

    // Buat struct value
    input := entity.RegisterInput{
        Username: req.Username,
        Email:  req.Email,
        FullName: req.FullName,
        Password: req.Password,
    }

    // FIX: Mengirim alamat memori (&input)
    userResp, err := h.authService.Register(&input) 
    
    if err != nil {
        switch err {
        case service.ErrUsernameTaken,
             service.ErrEmailTaken:
             c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
        default:
             c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        }
        return
    }

    c.JSON(http.StatusCreated, userResp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	resp, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case service.ErrInactiveAccount:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	newToken, err := h.authService.Refresh(req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrInvalidRefreshToken,
			service.ErrInvalidUserID:
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": newToken})
}

func (h *AuthHandler) Profile(c *gin.Context) {
	// asumsi middleware JWT kamu set "user_id" di context
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := rawID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id in context"})
		return
	}

	resp, err := h.authService.GetProfile(userID)
	if err != nil {
		if err == service.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

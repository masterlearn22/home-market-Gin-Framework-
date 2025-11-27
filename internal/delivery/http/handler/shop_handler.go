package handler

import (
	"net/http"

	entity "home-market/internal/domain"
	service "home-market/internal/service/postgresql"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShopHandler struct {
	shopService *service.ShopService
}

func NewShopHandler(shopService *service.ShopService) *ShopHandler {
	return &ShopHandler{shopService: shopService}
}

func (h *ShopHandler) CreateShop(c *gin.Context) {
	// ambil user id dari middleware JWT
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := rawID.(uuid.UUID)

	// ambil role dari middleware
	roleRaw, ok := c.Get("role_name")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "role missing"})
		return
	}
	role := roleRaw.(string)

	// bind input
	var input entity.CreateShopInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
		return
	}

	shop, err := h.shopService.CreateShop(userID, role, input)
	if err != nil {
		switch err {
		case service.ErrNotSeller:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case service.ErrShopExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, shop)
}

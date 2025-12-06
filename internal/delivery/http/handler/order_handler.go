package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	service "home-market/internal/service/postgresql"
	entity "home-market/internal/domain"
)

type OrderHandler struct {
	orderService *service.OrderService 
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// FR-BUYER-01 & FR-BUYER-02: Melihat & Filter Marketplace (GET /market/items)
func (h *OrderHandler) GetMarketplaceItems(c *gin.Context) {
	var filter entity.ItemFilter
	
	// c.ShouldBindQuery dapat menangani ItemFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query parameters", "detail": err.Error()})
		return
	}

	items, err := h.orderService.GetMarketplaceItems(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

// FR-BUYER-03: Melihat Detail Barang (GET /market/items/:id)
func (h *OrderHandler) GetItemDetail(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	item, err := h.orderService.GetItemDetail(itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": item})
}

// FR-BUYER-04: Membuat Order (POST /orders)
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)
	
	if role != "buyer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: only Buyer can create orders"})
		return
	}

	var input entity.CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input", "detail": err.Error()})
		return
	}

	order, err := h.orderService.CreateOrder(userID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Order created successfully", "order": order})
}

// FR-ORDER-02: Update Status Order (PATCH /orders/:id/status)
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var input entity.UpdateOrderStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input for status", "detail": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	order, err := h.orderService.UpdateOrderStatus(userID, role, orderID, input.NewStatus) // Asumsi service menerima string status
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order status updated", "order": order})
}

// FR-ORDER-03: Input Nomor Resi Pengiriman (POST /orders/:id/shipping)
func (h *OrderHandler) InputShippingReceipt(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var input entity.InputShippingReceiptInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input for shipping", "detail": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	order, err := h.orderService.InputShippingReceipt(userID, role, orderID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "shipping receipt input successfully, status changed to shipped", "order": order})
}

// FR-ORDER-04: Tracking Order (Buyer) (GET /orders/:id/tracking)
func (h *OrderHandler) GetOrderTracking(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	order, items, err := h.orderService.GetOrderTracking(userID, role, orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": order,
		"items": items,
	})
}
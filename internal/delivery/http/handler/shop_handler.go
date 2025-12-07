package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	entity "home-market/internal/domain"
	service "home-market/internal/service/postgresql" // Asumsi service gabungan ada di sini
)

// ShopItemHandler menangani semua operasi Seller: Shop Setup, Category Management, dan Item CRUD.
type ShopItemHandler struct {
	// Dependency tunggal: Service gabungan
	shopItemService *service.ShopItemService 
}

func NewShopItemHandler(shopItemService *service.ShopItemService) *ShopItemHandler {
	return &ShopItemHandler{shopItemService: shopItemService}
}

// ===============================================
// 1. SHOP SETUP METHODS (dari ShopHandler)
// ===============================================

func (h *ShopItemHandler) CreateShop(c *gin.Context) {
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := rawID.(uuid.UUID)

	roleRaw, ok := c.Get("role_name")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "role missing"})
		return
	}
	role := roleRaw.(string)

	var input entity.CreateShopInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
		return
	}

	// Memanggil service gabungan
	shop, err := h.shopItemService.CreateShop(userID, role, input)
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

// ===============================================
// 2. CATEGORY MANAGEMENT METHODS (dari CategoryHandler)
// ===============================================

func (h *ShopItemHandler) CreateCategory(c *gin.Context) {
	rawID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := rawID.(uuid.UUID)

	rawRole, ok := c.Get("role_name")
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "role missing"})
		return
	}
	role := rawRole.(string)

	var input entity.CreateCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "detail": err.Error()})
		return
	}

	// Memanggil service gabungan
	category, err := h.shopItemService.CreateCategory(userID, role, input)
	if err != nil {
		switch err {
		case service.ErrNotSeller:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case service.ErrNoShopOwned:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case service.ErrCategoryExists:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, category)
}

// ===============================================
// 3. ITEM CRUD METHODS (dari ItemHandler)
// ===============================================

func (h *ShopItemHandler) CreateItem(c *gin.Context) {
	fmt.Println("Content-Type:", c.GetHeader("Content-Type"))

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	// --- FORM MULTIPART ---
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form-data", "detail": err.Error()})
		return
	}

	// helper
	get := func(key string) string {
		if v, ok := form.Value[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}

	// --- Ambil text fields ---
	name := get("name")
	description := get("description")
	priceStr := get("price")
	stockStr := get("stock")
	condition := get("condition")
	categoryIDStr := get("category_id")

	if name == "" || priceStr == "" || stockStr == "" || categoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid price"})
		return
	}

	stock, err := strconv.Atoi(stockStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid stock"})
		return
	}

	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
		return
	}

	input := entity.CreateItemInput{
		Name: name,
		Description: description,
		Price: price,
		Stock: stock,
		Condition: condition,
		CategoryID: categoryID,
	}

	// --- Images ---
	files := form.File["images"]

	var imageURLs []string
	for _, file := range files {
		filename := uuid.New().String() + filepath.Ext(file.Filename)
		savePath := "uploads/items/" + filename

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		imageURLs = append(imageURLs, "/uploads/items/"+filename)
	}

	// --- Service ---
	item, images, err := h.shopItemService.CreateItem(userID, role, input, imageURLs) // Memanggil service gabungan
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"item": item,
		"images": images,
	})
}

func (h *ShopItemHandler) UpdateItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var input entity.UpdateItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input", "detail": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	updatedItem, err := h.shopItemService.UpdateItem(userID, itemID, input) // Memanggil service gabungan
	if err != nil {
		// Asumsi error handling di service sudah mencakup unauthorized/not found
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item updated", "data": updatedItem})
}

func (h *ShopItemHandler) DeleteItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.shopItemService.DeleteItem(userID, itemID); err != nil { // Memanggil service gabungan
		// Asumsi error handling di service sudah mencakup unauthorized/not found
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item archived/deleted successfully"})
}
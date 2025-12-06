// [internal/delivery/http/handler/item_handler.go]

package handler

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	service "home-market/internal/service/postgresql" // Asumsi service masih di sini
	entity "home-market/internal/domain"             // Asumsi entity diimpor
)

// ItemHandler: Core Item CRUD (FR-SELLER-03, 04, 05, 06)
type ItemHandler struct {
	itemService *service.ItemService
}

func NewItemHandler(itemService *service.ItemService) *ItemHandler {
	return &ItemHandler{itemService: itemService}
}

// FR-SELLER-03: CreateItem
func (h *ItemHandler) CreateItem(c *gin.Context) {
	fmt.Println("Content-Type:", c.GetHeader("Content-Type"))

	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	// --- FORM MULTIPART ---
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid form-data", "detail": err.Error()})
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
		c.JSON(400, gin.H{"error": "missing required fields"})
		return
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid price"})
		return
	}

	stock, err := strconv.Atoi(stockStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid stock"})
		return
	}

	categoryID, err := uuid.Parse(categoryIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid category_id"})
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
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		imageURLs = append(imageURLs, "/uploads/items/"+filename)
	}

	// --- Service ---
	item, images, err := h.itemService.CreateItem(userID, role, input, imageURLs)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"item": item,
		"images": images,
	})
}

// FR-SELLER-04 & FR-SELLER-06: UpdateItem
func (h *ItemHandler) UpdateItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid item id"})
		return
	}

	var input entity.UpdateItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": "invalid input", "detail": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	updatedItem, err := h.itemService.UpdateItem(userID, itemID, input)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "item updated", "data": updatedItem})
}

// FR-SELLER-05: DeleteItem (Soft Delete)
func (h *ItemHandler) DeleteItem(c *gin.Context) {
	idStr := c.Param("id")
	itemID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid item id"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.itemService.DeleteItem(userID, itemID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "item archived/deleted successfully"})
}
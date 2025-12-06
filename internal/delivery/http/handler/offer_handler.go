package handler

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	service "home-market/internal/service/postgresql"
	entity "home-market/internal/domain"
)

type OfferHandler struct {
	offerService *service.OfferService 
}

func NewOfferHandler(offerService *service.OfferService) *OfferHandler {
	return &OfferHandler{offerService: offerService}
}

// FR-GIVER-01 & FR-GIVER-02: Membuat Penawaran (POST /offers)
func (h *OfferHandler) CreateOffer(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	if role != "giver" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: only Giver can create offers"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form-data", "detail": err.Error()})
		return
	}

	// Helper untuk mengambil nilai form
	get := func(key string) string {
		if v, ok := form.Value[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}

	// Mapping input manual dari form
	input := entity.CreateOfferInput{
		SellerIDStr:   get("seller_id"),
		ItemName:      get("item_name"),
		Description:   get("description"),
		Condition:     get("condition"),
		Location:      get("location"),
	}
	
	// Parse Expected Price
	priceStr := get("expected_price")
	input.ExpectedPrice, err = strconv.ParseFloat(priceStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expected_price"})
		return
	}

	// --- Images Upload (FR-GIVER-02) ---
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}
	
	file := files[0]
	filename := uuid.New().String() + filepath.Ext(file.Filename)
	savePath := "uploads/offers/" + filename 
	
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	imageURL := "/uploads/offers/" + filename
	
	// --- Service Call ---
	offer, err := h.offerService.CreateOffer(userID, role, input, imageURL) // Panggil OfferService
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Offer created successfully. Waiting for seller response.",
		"offer": offer,
	})
}

// FR-GIVER-03: Melihat Status Penawaran (GET /offers/my)
func (h *OfferHandler) GetMyOffers(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	if role != "giver" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: only Giver can view offers"})
		return
	}

	offers, err := h.offerService.GetMyOffers(userID, role) // Panggil OfferService
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"offers": offers})
}

// FR-OFFER-01: Seller Melihat Penawaran (GET /offers/inbox)
func (h *OfferHandler) GetOffersToSeller(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("role_name").(string)

	offers, err := h.offerService.GetOffersToSeller(userID, role) // Panggil OfferService
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"offers": offers})
}

// FR-OFFER-02: Seller Menerima Penawaran (POST /offers/:id/accept)
func (h *OfferHandler) AcceptOffer(c *gin.Context) {
	offerIDStr := c.Param("id")
	offerID, err := uuid.Parse(offerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offer id"})
		return
	}

	var input entity.AcceptOfferInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input for agreed price", "detail": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	offer, draftItem, err := h.offerService.AcceptOffer(userID, offerID, input) // Panggil OfferService
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Offer accepted successfully. Draft item created.",
		"offer": offer,
		"draft_item": draftItem, 
	})
}

// FR-OFFER-03: Seller Menolak Penawaran (POST /offers/:id/reject)
func (h *OfferHandler) RejectOffer(c *gin.Context) {
	offerIDStr := c.Param("id")
	offerID, err := uuid.Parse(offerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offer id"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	offer, err := h.offerService.RejectOffer(userID, offerID) // Panggil OfferService
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Offer rejected successfully.",
		"offer": offer,
	})
}
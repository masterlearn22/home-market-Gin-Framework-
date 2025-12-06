// [internal/service/offer_service.go]

package service

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
	"log"

	entity "home-market/internal/domain"
	mongorepo "home-market/internal/repository/mongodb"
	repo "home-market/internal/repository/postgresql"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// --- ERROR DEFINITIONS ---
var (
	ErrNotGiver         = errors.New("access denied: only giver role is allowed")
	ErrOfferNotFound    = errors.New("offer not found")
	ErrOfferStatus      = errors.New("offer is not in pending status")
	ErrNotSellerOrOwner = errors.New("unauthorized: access denied or you are not the owner")
)

// --- OFFER SERVICE STRUCT ---
type OfferService struct {
	offerRepo repo.OfferRepository
	itemRepo  repo.ItemRepository
	shopRepo  repo.ShopRepository
	logRepo   mongorepo.LogRepository // Untuk notifikasi/history
}

func NewOfferService(offerRepo repo.OfferRepository, itemRepo repo.ItemRepository, shopRepo repo.ShopRepository, logRepo mongorepo.LogRepository) *OfferService {
	return &OfferService{
		offerRepo: offerRepo,
		itemRepo:  itemRepo,
		shopRepo:  shopRepo,
		logRepo:   logRepo,
	}
}

// --- HELPER FUNCTIONS ---

// Helper untuk validasi kepemilikan
func (s *OfferService) checkSellerOwnership(userID uuid.UUID) (*entity.Shop, error) {
	shop, err := s.shopRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	return shop, nil
}

// FR-OFFER-04: Helper untuk membuat Item Draft
func (s *OfferService) createDraftItemFromOffer(offer *entity.Offer, shopID uuid.UUID) *entity.Item {
	return &entity.Item{
		ID:uuid.New(),
		ShopID:shopID,
		CategoryID:uuid.Nil, 
		Name:offer.ItemName,
		Description: fmt.Sprintf("Draft dari Penawaran: %s. Kondisi: %s. Lokasi Awal: %s.", offer.Description, offer.Condition, offer.Location),
		Price: offer.AgreedPrice.Float64, 
		Stock: 1,
		Condition: offer.Condition,
		Status:"draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// FR-NOTIF-01/02/03: Helper untuk membuat dan menyimpan notifikasi
func (s *OfferService) createAndSaveNotification(userID uuid.UUID, title string, message string, notiType string, relatedID uuid.UUID) {
    noti := &entity.Notification{
        ID: primitive.NewObjectID(),
        UserID: userID,
        Title: title,
        Message: message,
        Type: notiType,
        RelatedID: relatedID,
        IsRead: false,
        CreatedAt: time.Now(),
    }
    
    if err := s.logRepo.SaveNotification(noti); err != nil {
        log.Printf("Warning: failed to save notification for user %s: %v", userID.String(), err)
    }
}


// --- OFFER SERVICE METHODS ---

// FR-GIVER-01 & FR-GIVER-02: Membuat Penawaran Barang
func (s *OfferService) CreateOffer(userID uuid.UUID, role string, input entity.CreateOfferInput, imageURL string) (*entity.Offer, error) {
	if role != "giver" {
		return nil, ErrNotGiver
	}

	var sellerID uuid.UUID
	if input.SellerIDStr != "" {
		id, err := uuid.Parse(input.SellerIDStr)
		if err != nil {
			return nil, errors.New("invalid seller_id format")
		}
		sellerID = id
	}

	if input.ExpectedPrice < 0 {
		return nil, errors.New("expected price cannot be negative")
	}

	offer := &entity.Offer{
		ID:uuid.New(),
		GiverID: userID,
		SellerID:sellerID,
		ItemName:input.ItemName,
		Description: input.Description,
		ImageURL:imageURL, 
		ExpectedPrice: input.ExpectedPrice,
		Condition: input.Condition,
		Location:input.Location,
		Status:"pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
    
	if err := s.offerRepo.CreateOffer(offer); err != nil { // Call OfferRepo
		return nil, err
	}

	// FR-NOTIF-01: Trigger notifikasi ke Seller
	if offer.SellerID != uuid.Nil {
        s.createAndSaveNotification(offer.SellerID, "Penawaran Baru Masuk", fmt.Sprintf("Anda menerima penawaran dari Giver untuk barang '%s'.", offer.ItemName), "offer", offer.ID)
	}

	return offer, nil
}

// FR-GIVER-03: Melihat Status Penawaran
func (s *OfferService) GetMyOffers(userID uuid.UUID, role string) ([]entity.Offer, error) {
	if role != "giver" {
		return nil, ErrNotGiver
	}
	return s.offerRepo.GetOffersByGiverID(userID) // Call OfferRepo
}

// FR-OFFER-01: Seller Melihat Penawaran
func (s *OfferService) GetOffersToSeller(userID uuid.UUID, role string) ([]entity.Offer, error) {
	if role != "seller" {
		return nil, errors.New("access denied: only seller can view offers")
	}

	shop, err := s.shopRepo.GetByUserID(userID) // Call ShopRepo
	if err != nil {
		return nil, err
	}
	if shop == nil {
		return nil, ErrNoShopOwned
	}
	return s.offerRepo.GetOffersBySellerID(userID) // Call OfferRepo
}

// FR-OFFER-02: Seller Menerima Penawaran
func (s *OfferService) AcceptOffer(userID uuid.UUID, offerID uuid.UUID, input entity.AcceptOfferInput) (*entity.Offer, *entity.Item, error) {
	var shop *entity.Shop 

	shop, err := s.checkSellerOwnership(userID) // Call Helper
	if err != nil {
		return nil, nil, err
	}
	if shop == nil {
		return nil, nil, ErrNoShopOwned
	}

	offer, err := s.offerRepo.GetOfferByID(offerID) // Call OfferRepo
	if err != nil {
		return nil, nil, err
	}
	if offer == nil {
		return nil, nil, ErrOfferNotFound
	}

	if offer.SellerID != uuid.Nil && offer.SellerID != userID {
		return nil, nil, ErrNotSellerOrOwner
	}

	if offer.Status != "pending" {
		return nil, nil, ErrOfferStatus
	}

	oldStatus := offer.Status // Simpan status lama

	// 4. Update Status dan Harga
	offer.Status = "accepted"
	offer.AgreedPrice = sql.NullFloat64{Float64: input.AgreedPrice, Valid: true} // sql.NullFloat64 dari import database/sql
    
	if err := s.offerRepo.UpdateOffer(offer); err != nil { // Call OfferRepo
		return nil, nil, err
	}

	// 5. Buat Draft Item (FR-OFFER-04)
	draftItem := s.createDraftItemFromOffer(offer, shop.ID)
	if err := s.itemRepo.CreateItem(draftItem); err != nil { // Call ItemRepo
		return offer, nil, errors.New("offer accepted, but failed to create draft item")
	}
    
    // Simpan History Status (FR-OFFER-02)
    history := &entity.HistoryStatus{
        ID: primitive.NewObjectID(), 
        RelatedID: offerID.String(), 
        RelatedType: "offer",
        OldStatus: oldStatus,
        NewStatus: "accepted",
        ChangedBy: userID.String(), 
        Timestamp: time.Now(),
    }
    if err := s.logRepo.SaveHistoryStatus(history); err != nil {
        log.Printf("Warning: failed to save history status for offer %s: %v", offerID.String(), err)
    }

	return offer, draftItem, nil
}

// FR-OFFER-03: Seller Menolak Penawaran
func (s *OfferService) RejectOffer(userID uuid.UUID, offerID uuid.UUID) (*entity.Offer, error) {
	if shop, err := s.checkSellerOwnership(userID); err != nil { // Call Helper
		return nil, err
	} else if shop == nil {
		return nil, ErrNoShopOwned
	}

	offer, err := s.offerRepo.GetOfferByID(offerID) // Call OfferRepo
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, ErrOfferNotFound
	}
	if offer.Status != "pending" {
		return nil, ErrOfferStatus
	}

	oldStatus := offer.Status // Simpan status lama

	// 2. Update Status (FR-OFFER-03)
	offer.Status = "rejected"

	if err := s.offerRepo.UpdateOffer(offer); err != nil { // Call OfferRepo
		return nil, err
	}
    
    // Simpan History Status (FR-OFFER-03)
    history := &entity.HistoryStatus{
        ID: primitive.NewObjectID(), 
        RelatedID: offerID.String(), 
        RelatedType: "offer",
        OldStatus: oldStatus,
        NewStatus: "rejected",
        ChangedBy: userID.String(), 
        Timestamp: time.Now(),
    }
    if err := s.logRepo.SaveHistoryStatus(history); err != nil {
        log.Printf("Warning: failed to save history status for offer %s: %v", offerID.String(), err)
    }

	return offer, nil
}
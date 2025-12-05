package service

import (
	"errors"
	"time"
    "fmt"
    "database/sql"
	entity "home-market/internal/domain"
	repo "home-market/internal/repository/postgresql"

	"github.com/google/uuid"
)

var (
	ErrInvalidStock      = errors.New("stock must be >= 0")
	ErrInvalidPrice      = errors.New("price must be >= 0")
	ErrCategoryNotOwned  = errors.New("category does not belong to seller's shop")
	ErrNotGiver = errors.New("access denied: only giver role is allowed")
    ErrOfferNotFound    = errors.New("offer not found")
    ErrOfferStatus      = errors.New("offer is not in pending status")
    ErrNotSellerOrOwner = errors.New("unauthorized: access denied or you are not the owner")
)

type ItemService struct {
	itemRepo repo.ItemRepository
}

func NewItemService(itemRepo repo.ItemRepository) *ItemService {
	return &ItemService{
		itemRepo: itemRepo,
	}
}

func (s *ItemService) CreateItem(userID uuid.UUID, role string, input entity.CreateItemInput, imageURLs []string) (*entity.Item, []entity.ItemImage, error) {

	if role != "seller" {
		return nil, nil, ErrNotSeller
	}

	shop, err := s.itemRepo.GetShopByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	if shop == nil {
		return nil, nil, ErrNoShopOwned
	}

	// Validasi kategori milik shop
	owned, err := s.itemRepo.IsCategoryOwnedByShop(input.CategoryID, shop.ID)
	if err != nil {
		return nil, nil, err
	}
	if !owned {
		return nil, nil, ErrCategoryNotOwned
	}

	if input.Stock < 0 {
		return nil, nil, ErrInvalidStock
	}
	if input.Price < 0 {
		return nil, nil, ErrInvalidPrice
	}

	item := &entity.Item{
		ID:          uuid.New(),
		ShopID:      shop.ID,
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Stock:       input.Stock,
		Condition:   input.Condition,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Simpan item
	if err := s.itemRepo.CreateItem(item); err != nil {
		return nil, nil, err
	}

	// Simpan gambar
	var images []entity.ItemImage
	for _, url := range imageURLs {
		img := entity.ItemImage{
			ID:       uuid.New(),
			ItemID:   item.ID,
			ImageURL: url,
			CreatedAt: time.Now(),
		}

		if err := s.itemRepo.CreateItemImage(&img); err != nil {
			return item, nil, err
		}

		images = append(images, img)
	}

	return item, images, nil
}

func (s *ItemService) UpdateItem(userID uuid.UUID, itemID uuid.UUID, input entity.UpdateItemInput) (*entity.Item, error) {
    // 1. Cek apakah Item ada
    item, err := s.itemRepo.GetItemByID(itemID)
    if err != nil {
        return nil, err
    }
    if item == nil {
        return nil, errors.New("item not found")
    }

    // 2. Cek Shop milik User
    shop, err := s.itemRepo.GetShopByUserID(userID)
    if err != nil {
        return nil, err
    }
    if shop == nil {
        return nil, errors.New("you do not have a shop")
    }

    // 3. Validasi Kepemilikan: Item.ShopID harus sama dengan Shop.ID milik user
    if item.ShopID != shop.ID {
        return nil, errors.New("unauthorized: this item does not belong to your shop")
    }

    // 4. Update Field (FR-SELLER-04 & FR-SELLER-06)
    item.Name = input.Name
    item.Description = input.Description
    item.Price = input.Price
    item.Stock = input.Stock
    item.Condition = input.Condition
    
    // Jika user mengirim status (misal mau re-activate), pakai itu. Jika kosong, biarkan yang lama.
    if input.Status != "" {
        item.Status = input.Status
    }

    // 5. Simpan ke DB
    if err := s.itemRepo.UpdateItem(item); err != nil {
        return nil, err
    }

    return item, nil
}


func (s *ItemService) DeleteItem(userID uuid.UUID, itemID uuid.UUID) error {
    // 1. Cek Item & Shop (Logic sama seperti update)
    item, err := s.itemRepo.GetItemByID(itemID)
    if err != nil {
        return err
    }
    if item == nil {
        return errors.New("item not found")
    }

    shop, err := s.itemRepo.GetShopByUserID(userID)
    if err != nil {
        return err
    }
    if shop == nil || item.ShopID != shop.ID {
        return errors.New("unauthorized")
    }

   
    item.Status = "inactive"

    // 3. Simpan perubahan
    return s.itemRepo.UpdateItem(item)
}

// FR-GIVER-01 & FR-GIVER-02: Membuat Penawaran Barang
func (s *ItemService) CreateOffer(userID uuid.UUID, role string, input entity.CreateOfferInput, imageURL string) (*entity.Offer, error) {
    if role != "giver" {
        return nil, ErrNotGiver // Validasi FR-GIVER-01
    }

    var sellerID uuid.UUID
    if input.SellerIDStr != "" {
        id, err := uuid.Parse(input.SellerIDStr)
        if err != nil {
            return nil, errors.New("invalid seller_id format")
        }
        // Opsional: Cek apakah SellerID valid dan memiliki toko aktif (tambahan validasi)
        sellerID = id
    }
    
    // Validasi dasar
    if input.ExpectedPrice < 0 {
        return nil, errors.New("expected price cannot be negative")
    }

    offer := &entity.Offer{
        ID:             uuid.New(),
        GiverID:        userID,
        SellerID:       sellerID,
        ItemName:       input.ItemName,
        Description:    input.Description,
        ImageURL:       imageURL, // URL dari file yang di-upload (FR-GIVER-02)
        ExpectedPrice:  input.ExpectedPrice,
        Condition:      input.Condition,
        Location:       input.Location,
        Status:         "pending", // Status awal selalu pending (FR-GIVER-01)
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }

    if err := s.itemRepo.CreateOffer(offer); err != nil {
        return nil, err
    }
    
    // Opsional: Trigger notifikasi ke Seller jika SellerID ada (FR-NOTIF-01)
    
    return offer, nil
}

// FR-GIVER-03: Melihat Status Penawaran
func (s *ItemService) GetMyOffers(userID uuid.UUID, role string) ([]entity.Offer, error) {
    if role != "giver" {
        return nil, ErrNotGiver
    }
    
    return s.itemRepo.GetOffersByGiverID(userID)
}

// FR-OFFER-01: Seller Melihat Penawaran
func (s *ItemService) GetOffersToSeller(userID uuid.UUID, role string) ([]entity.Offer, error) {
    if role != "seller" {
        return nil, errors.New("access denied: only seller can view offers")
    }

    // Pastikan user memiliki shop untuk menerima penawaran
    shop, err := s.itemRepo.GetShopByUserID(userID)
    if err != nil {
        return nil, err
    }
    if shop == nil {
        return nil, ErrNoShopOwned
    }

    // Mengambil offer berdasarkan UserID (SellerID)
    return s.itemRepo.GetOffersBySellerID(userID)
}

// FR-OFFER-02: Seller Menerima Penawaran
func (s *ItemService) AcceptOffer(userID uuid.UUID, offerID uuid.UUID, input entity.AcceptOfferInput) (*entity.Offer, *entity.Item, error) {
    var shop *entity.Shop // FIX 2: Deklarasikan 'shop' di luar if block

    // 1. Validasi Role dan Ownership: Cek User punya Shop
    shop, err := s.checkSellerOwnership(userID) // Panggil helper
    if err != nil {
        return nil, nil, err
    }
    if shop == nil {
        return nil, nil, ErrNoShopOwned 
    }

    offer, err := s.itemRepo.GetOfferByID(offerID)
    if err != nil {
        return nil, nil, err
    }
    if offer == nil {
        return nil, nil, ErrOfferNotFound
    }
    
    // 2. Ownership check pada Offer
    // FIX 1: Ganti IsZero() dengan uuid.Nil
    if offer.SellerID != uuid.Nil && offer.SellerID != userID { 
        return nil, nil, ErrNotSellerOrOwner
    }

    // 3. Cek Status Offer
    if offer.Status != "pending" {
        return nil, nil, ErrOfferStatus
    }

    // 4. Update Status dan Harga
    offer.Status = "accepted"
    offer.AgreedPrice = sql.NullFloat64{Float64: input.AgreedPrice, Valid: true}
    
    if err := s.itemRepo.UpdateOffer(offer); err != nil {
        return nil, nil, err
    }

    // 5. Buat Draft Item (shop sekarang bisa diakses)
    draftItem := s.createDraftItemFromOffer(offer, shop.ID) 
    if err := s.itemRepo.CreateItem(draftItem); err != nil {
        return offer, nil, errors.New("offer accepted, but failed to create draft item")
    }

    return offer, draftItem, nil
}

// FR-OFFER-03: Seller Menolak Penawaran
func (s *ItemService) RejectOffer(userID uuid.UUID, offerID uuid.UUID) (*entity.Offer, error) {
    // 1. Validasi Role dan Ownership
    if shop, err := s.checkSellerOwnership(userID); err != nil {
        return nil, err
    } else if shop == nil {
        return nil, ErrNoShopOwned
    }

    offer, err := s.itemRepo.GetOfferByID(offerID)
    if err != nil {
        return nil, err
    }
    if offer == nil {
        return nil, ErrOfferNotFound
    }
    if offer.Status != "pending" {
        return nil, ErrOfferStatus
    }

    // 2. Update Status (FR-OFFER-03)
    offer.Status = "rejected"
    // AgreedPrice tidak perlu diubah/di-set

    if err := s.itemRepo.UpdateOffer(offer); err != nil {
        return nil, err
    }

    // Opsional: Simpan ke history_status (Mongo)

    return offer, nil
}

// --- Helper Functions ---

// Helper untuk validasi kepemilikan
func (s *ItemService) checkSellerOwnership(userID uuid.UUID) (*entity.Shop, error) {
    shop, err := s.itemRepo.GetShopByUserID(userID)
    if err != nil {
        return nil, err
    }
    // Jika shop == nil, akan dikembalikan sebagai (nil, nil)
    return shop, nil
}

// FR-OFFER-04: Helper untuk membuat Item Draft
func (s *ItemService) createDraftItemFromOffer(offer *entity.Offer, shopID uuid.UUID) *entity.Item {
    return &entity.Item{
        ID:          uuid.New(),
        ShopID:      shopID,
        CategoryID:  uuid.Nil, // Default: Item draft tidak punya kategori sebelum diedit Seller
        Name:        offer.ItemName,
        Description: fmt.Sprintf("Draft dari Penawaran: %s. Kondisi: %s. Lokasi Awal: %s.", offer.Description, offer.Condition, offer.Location),
        Price:       offer.AgreedPrice.Float64, // Menggunakan harga kesepakatan
        Stock:       1,                         // Stok awal 1 unit
        Condition:   offer.Condition,
        Status:      "draft",                   // FR-OFFER-04: Status draft/inactive
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }
}

// [internal/service/item_service.go]

// FR-BUYER-01 & FR-BUYER-02: Melihat & Filter Marketplace
func (s *ItemService) GetMarketplaceItems(filter entity.ItemFilter) ([]entity.Item, error) {
    // Asumsi filter role sudah dilakukan di middleware (jika perlu)
    return s.itemRepo.GetMarketItems(filter)
}

// FR-BUYER-03: Melihat Detail Barang
func (s *ItemService) GetItemDetail(itemID uuid.UUID) (*entity.Item, error) {
    // Implementasi ini mengandalkan GetItemForOrder/GetItemByID
    item, err := s.itemRepo.GetItemForOrder(itemID)
    if err != nil {
        return nil, err
    }
    if item == nil || item.Status != "active" {
        return nil, errors.New("item not found or inactive")
    }
    // Asumsi JOIN dengan data Shop/Images dilakukan di layer ini atau di handler
    return item, nil
}

// FR-BUYER-0ER-04: Membuat Order
func (s *ItemService) CreateOrder(buyerID uuid.UUID, input entity.CreateOrderInput) (*entity.Order, error) {
    // 1. Validasi Stok, Harga, dan Grouping per Toko
    
    // Map untuk mengelompokkan item berdasarkan ShopID
    shopItems := make(map[uuid.UUID][]entity.OrderItem)
    itemDetails := make(map[uuid.UUID]*entity.Item)
    
    // Looping validasi
    for _, itemInput := range input.Items {
        item, err := s.itemRepo.GetItemForOrder(itemInput.ItemID)
        if err != nil {
            return nil, errors.New("database error during item fetch")
        }
        if item == nil || item.Status != "active" || item.Stock < itemInput.Quantity {
            return nil, errors.New("invalid item, insufficient stock, or item inactive") // Validasi stok tersedia
        }
        
        // Simpan detail item dan group ke shop
        itemDetails[item.ID] = item
        
        orderItem := entity.OrderItem{
            ItemID: item.ID,
            Quantity: itemInput.Quantity,
            Price: item.Price, // Harga saat order dibuat (snapshot)
            OrderID: uuid.Nil, // Akan diisi saat order dibuat
        }
        shopItems[item.ShopID] = append(shopItems[item.ShopID], orderItem)
    }
    
    // 2. Membuat Order per Toko (Simplifikasi: Hanya support satu toko per order/per transaksi ini)
    if len(shopItems) != 1 {
        // Dalam konteks nyata, harus ada logic keranjang/cart multi-shop atau multiple orders.
        return nil, errors.New("multi-shop orders are not supported in a single transaction yet")
    }
    
    var shopID uuid.UUID
    var itemsForOrder []entity.OrderItem
    var totalPrice float64
    
    for id, items := range shopItems {
        shopID = id
        itemsForOrder = items
        for _, item := range items {
            totalPrice += item.Price * float64(item.Quantity)
        }
        break // Ambil hanya toko pertama (karena validasi di atas)
    }

    // 3. Buat Struct Order
    order := &entity.Order{
        ID:              uuid.New(),
        BuyerID:         buyerID,
        ShopID:          shopID,
        TotalPrice:      totalPrice,
        Status:          "pending", // Default status
        ShippingAddress: input.ShippingAddress,
        ShippingCourier: input.ShippingCourier,
    }
    
    // Update OrderID di OrderItem structs
    for i := range itemsForOrder {
        itemsForOrder[i].OrderID = order.ID
        itemsForOrder[i].ID = uuid.New()
    }

    // 4. Jalankan Transaksi
    if err := s.itemRepo.CreateOrderTransaction(order, itemsForOrder); err != nil {
        return nil, err
    }

    return order, nil
}
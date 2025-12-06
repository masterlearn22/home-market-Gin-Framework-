package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	entity "home-market/internal/domain"
	mongorepo "home-market/internal/repository/mongodb"
	repo "home-market/internal/repository/postgresql"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)


type OrderService struct {
	orderRepo repo.OrderRepository
	shopRepo  repo.ShopRepository 
	logRepo   mongorepo.LogRepository 
}

func NewOrderService(orderRepo repo.OrderRepository, shopRepo repo.ShopRepository, logRepo mongorepo.LogRepository) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		shopRepo: shopRepo,
		logRepo: logRepo,
	}
}

// Helper function untuk membuat dan menyimpan notifikasi (FR-NOTIF)
func (s *OrderService) createAndSaveNotification(userID uuid.UUID, title string, message string, notiType string, relatedID uuid.UUID) {
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

// FR-BUYER-01 & FR-BUYER-02: Melihat & Filter Marketplace
func (s *OrderService) GetMarketplaceItems(filter entity.ItemFilter) ([]entity.Item, error) {
	return s.orderRepo.GetMarketItems(filter)
}

// FR-BUYER-03: Melihat Detail Barang
func (s *OrderService) GetItemDetail(itemID uuid.UUID) (*entity.Item, error) {
	item, err := s.orderRepo.GetItemForOrder(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.Status != "active" {
		return nil, errors.New("item not found or inactive")
	}
	return item, nil
}

// FR-BUYER-04 & FR-NOTIF-03: Membuat Order
func (s *OrderService) CreateOrder(buyerID uuid.UUID, input entity.CreateOrderInput) (*entity.Order, error) {
	
	// Logika Validasi Stok, Harga, dan Grouping per Toko (Dipindahkan dari ItemService lama)
	shopItems := make(map[uuid.UUID][]entity.OrderItem)
	
	for _, itemInput := range input.Items {
		item, err := s.orderRepo.GetItemForOrder(itemInput.ItemID)
		if err != nil { return nil, errors.New("database error during item fetch") }
		if item == nil || item.Status != "active" || item.Stock < itemInput.Quantity {
			return nil, errors.New("invalid item, insufficient stock, or item inactive")
		}

		orderItem := entity.OrderItem{
			ItemID: item.ID, Quantity: itemInput.Quantity, Price: item.Price, OrderID: uuid.Nil,
		}
		shopItems[item.ShopID] = append(shopItems[item.ShopID], orderItem)
	}

	if len(shopItems) != 1 {
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
		break
	}
    
    // Ambil ID Pemilik Toko (ShopRepo)
    shopOwnerID, err := s.shopRepo.GetShopOwnerID(shopID) 
    if err != nil && err.Error() != "shop not found" {
        log.Printf("Warning: failed to retrieve shop owner ID for notification: %v", err)
    }

	// 3. Buat Struct Order
	order := &entity.Order{
		ID: uuid.New(), BuyerID: buyerID, ShopID: shopID, TotalPrice: totalPrice, Status: "pending", 
		ShippingAddress: input.ShippingAddress, ShippingCourier: input.ShippingCourier,
	}

	for i := range itemsForOrder {
		itemsForOrder[i].OrderID = order.ID
		itemsForOrder[i].ID = uuid.New()
	}

	// 4. Jalankan Transaksi
	if err := s.orderRepo.CreateOrderTransaction(order, itemsForOrder); err != nil {
		return nil, err
	}
    
    // --- Notification Trigger (FR-NOTIF-03) ---
    if shopOwnerID != uuid.Nil {
        s.createAndSaveNotification(
            shopOwnerID, "Order Baru Masuk",
            fmt.Sprintf("Anda menerima order baru #%s dengan total %.2f.", order.ID.String()[:8], order.TotalPrice),
            "new_order", order.ID,
        )
    }
    // --- End Notification Trigger ---


	return order, nil
}

// FR-ORDER-02 & FR-NOTIF-02: Update Status Order
func (s *OrderService) UpdateOrderStatus(userID uuid.UUID, role string, orderID uuid.UUID, status string) (*entity.Order, error) {
	if !ValidOrderStatuses[status] {
		return nil, errors.New("invalid status value")
	}
    
    order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil { return nil, err }
	if order == nil { return nil, errors.New("order not found") }

	// Logika otorisasi Seller/Admin (ShopRepo)
	shop, _ := s.shopRepo.GetByUserID(userID)
	isOwner := shop != nil && order.ShopID == shop.ID
	isAdmin := role == "admin"
	
	if !isOwner && !isAdmin {
		return nil, errors.New("unauthorized: you are not the shop owner or admin")
	}

	// oldStatus := order.Status

	// Update status (OrderRepo)
	if err := s.orderRepo.UpdateOrderStatus(orderID, status); err != nil { return nil, err }
    order.Status = status 

    // Simpan Riwayat Status (LogRepo)
	// Asumsi Anda akan memanggil s.logRepo.SaveHistoryStatus di sini
    
	// Trigger Notifikasi (FR-NOTIF-02)
    s.createAndSaveNotification(
        order.BuyerID, "Status Order Berubah",
        fmt.Sprintf("Status order Anda #%s telah diperbarui menjadi %s.", orderID.String()[:8], status),
        "order_status", order.ID,
    )

	return order, nil
}

// FR-ORDER-03 & FR-NOTIF-02: Input Nomor Resi Pengiriman
func (s *OrderService) InputShippingReceipt(userID uuid.UUID, role string, orderID uuid.UUID, input entity.InputShippingReceiptInput) (*entity.Order, error) {
    order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil { return nil, err }
	if order == nil { return nil, errors.New("order not found") }

	// Logika otorisasi Seller/Admin (ShopRepo)
	shop, _ := s.shopRepo.GetByUserID(userID)
	isOwner := shop != nil && order.ShopID == shop.ID
	isAdmin := role == "admin"
	
	if !isOwner && !isAdmin { return nil, errors.New("unauthorized") }

	// Update shipment (OrderRepo)
	if err := s.orderRepo.UpdateOrderShipment(orderID, input.ShippingCourier, input.ShippingReceipt); err != nil { return nil, err }
    order.ShippingCourier = input.ShippingCourier 
    order.ShippingReceipt = input.ShippingReceipt
    order.Status = "shipped"

    // Simpan Riwayat Status (LogRepo)
    
	// Trigger Notifikasi (FR-NOTIF-02)
    s.createAndSaveNotification(
        order.BuyerID, "Barang Anda Dikirim",
        fmt.Sprintf("Order Anda #%s telah dikirim dengan resi %s.", orderID.String()[:8], input.ShippingReceipt),
        "order_status", order.ID,
    )

	return order, nil
}

// FR-ORDER-04: Tracking Order (Buyer)
func (s *OrderService) GetOrderTracking(userID uuid.UUID, role string, orderID uuid.UUID) (*entity.Order, []entity.OrderItem, error) {
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil { return nil, nil, err }
	if order == nil { return nil, nil, errors.New("order not found") }

	// Logika otorisasi Buyer/Admin
	isAdmin := role == "admin"
	isBuyer := order.BuyerID == userID
	if !isBuyer && !isAdmin { return nil, nil, errors.New("unauthorized: access denied") }

	// Ambil order items (OrderRepo)
	items, err := s.orderRepo.GetOrderItems(orderID)
	if err != nil { return order, nil, err }
	
	// Tambahan: Ambil history status dari MongoDB

	return order, items, nil
}
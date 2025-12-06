package repository

import (
	"database/sql"
	"fmt"
	"strings"

	entity "home-market/internal/domain"
	"github.com/google/uuid"
)

// OrderRepository Interface (Dibutuhkan untuk ItemService/OrderService)
type OrderRepository interface {
	GetMarketItems(filter entity.ItemFilter) ([]entity.Item, error)
	GetItemForOrder(itemID uuid.UUID) (*entity.Item, error)
	CreateOrderTransaction(order *entity.Order, items []entity.OrderItem) error
	GetOrderByID(orderID uuid.UUID) (*entity.Order, error)
	UpdateOrderStatus(orderID uuid.UUID, status string) error
	UpdateOrderShipment(orderID uuid.UUID, courier string, receipt string) error
	GetOrderItems(orderID uuid.UUID) ([]entity.OrderItem, error)
}

// Struct koneksi untuk OrderRepository
type orderRepository struct {
	db *sql.DB
}

// Constructor WAJIB
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

// Helper untuk mengambil Item berdasarkan ID (digunakan oleh GetItemForOrder)
func (r *orderRepository) getItemByID(id uuid.UUID) (*entity.Item, error) {
	var item entity.Item
	query := `
		SELECT id, shop_id, category_id, name, description, price, stock, condition, status, created_at, updated_at
		FROM items WHERE id = $1
	`
	// Perhatikan: CategoryID di Item struct harus berupa sql.NullUUID jika boleh NULL
	// Jika CategoryID didefinisikan sebagai uuid.UUID di struct entity.Item,
	// maka harus ada penanganan khusus jika nilainya NULL di DB.
	err := r.db.QueryRow(query, id).Scan(
		&item.ID, &item.ShopID, &item.CategoryID, &item.Name, &item.Description,
		&item.Price, &item.Stock, &item.Condition, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &item, err
}

// FR-BUYER-01 & FR-BUYER-02: Melihat & Filter Marketplace
func (r *orderRepository) GetMarketItems(filter entity.ItemFilter) ([]entity.Item, error) {
	var items []entity.Item
	
	baseQuery := `
		SELECT id, shop_id, category_id, name, description, price, stock, condition, status, created_at, updated_at
		FROM items
		WHERE status = 'active' AND stock > 0
	`
	args := []interface{}{}
	whereClauses := []string{}
	
	// Implementasi Filter (FR-BUYER-02)
	if filter.Keyword != "" {
		keyword := fmt.Sprintf("%%%s%%", filter.Keyword)
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE '%s' OR description ILIKE '%s')", keyword, keyword))
	}
	if filter.CategoryID != uuid.Nil {
		whereClauses = append(whereClauses, fmt.Sprintf("category_id = '%s'", filter.CategoryID.String()))
	}
	if filter.MinPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price >= %f", filter.MinPrice))
	}
	if filter.MaxPrice > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("price <= %f", filter.MaxPrice))
	}

	if len(whereClauses) > 0 {
		baseQuery += " AND " + strings.Join(whereClauses, " AND ")
	}

	// Penambahan Pagination
	if filter.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		baseQuery += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}
    
	rows, err := r.db.Query(baseQuery, args...) 
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item entity.Item
		// Scan semua field (Asumsi struct entity.Item lengkap)
		err := rows.Scan(
			&item.ID, &item.ShopID, &item.CategoryID, &item.Name, &item.Description, &item.Price, 
			&item.Stock, &item.Condition, &item.Status, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// FR-BUYER-03 & FR-BUYER-04: Ambil Item Detail
func (r *orderRepository) GetItemForOrder(itemID uuid.UUID) (*entity.Item, error) {
	// Memanggil helper GetItemByID yang ada di repository ini.
	return r.getItemByID(itemID) 
}

// FR-BUYER-04: Membuat Order (Menggunakan Transaksi)
func (r *orderRepository) CreateOrderTransaction(order *entity.Order, orderItems []entity.OrderItem) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	
	// 1. Insert Order
	orderQuery := `
		INSERT INTO orders (id, buyer_id, shop_id, total_price, status, shipping_address, shipping_courier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	if _, err := tx.Exec(orderQuery, order.ID, order.BuyerID, order.ShopID, order.TotalPrice, order.Status, order.ShippingAddress, order.ShippingCourier); err != nil {
		tx.Rollback()
		return err
	}

	// 2. Insert Order Items & Update Stock (Loop)
	for _, item := range orderItems {
		// Insert Order Item
		itemQuery := `INSERT INTO order_items (id, order_id, item_id, quantity, price, created_at) VALUES ($1, $2, $3, $4, $5, NOW())`
		if _, err := tx.Exec(itemQuery, uuid.New(), item.OrderID, item.ItemID, item.Quantity, item.Price); err != nil {
			tx.Rollback()
			return err
		}
		
		// Update Stock (Decrement)
		stockQuery := `UPDATE items SET stock = stock - $1 WHERE id = $2`
		if _, err := tx.Exec(stockQuery, item.Quantity, item.ItemID); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *orderRepository) GetOrderByID(orderID uuid.UUID) (*entity.Order, error) {
	var order entity.Order
	query := `
		SELECT id, buyer_id, shop_id, total_price, status, shipping_address, shipping_courier, shipping_receipt, created_at, updated_at
		FROM orders WHERE id = $1
	`
	// Asumsi struct entity.Order lengkap
	err := r.db.QueryRow(query, orderID).Scan(
		&order.ID, &order.BuyerID, &order.ShopID, &order.TotalPrice, &order.Status, 
		&order.ShippingAddress, &order.ShippingCourier, &order.ShippingReceipt, &order.CreatedAt, &order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &order, err
}

// FR-ORDER-02: Update Status
func (r *orderRepository) UpdateOrderStatus(orderID uuid.UUID, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, status, orderID)
	return err
}

// FR-ORDER-03: Input Nomor Resi
func (r *orderRepository) UpdateOrderShipment(orderID uuid.UUID, courier string, receipt string) error {
	// FR-ORDER-03 juga mengubah status menjadi 'shipped'
	query := `
		UPDATE orders SET shipping_courier = $1, shipping_receipt = $2, status = 'shipped', updated_at = NOW() 
		WHERE id = $3
	`
	_, err := r.db.Exec(query, courier, receipt, orderID)
	return err
}

// FR-ORDER-04: Ambil Order Items untuk Tracking
func (r *orderRepository) GetOrderItems(orderID uuid.UUID) ([]entity.OrderItem, error) {
	var items []entity.OrderItem
	query := `
		SELECT id, order_id, item_id, quantity, price, created_at
		FROM order_items
		WHERE order_id = $1
	`
	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item entity.OrderItem
		// Asumsi struct entity.OrderItem lengkap
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.ItemID, &item.Quantity, &item.Price, &item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
package entity
import (
	"time"
	"github.com/google/uuid"
)

type Order struct {
	ID            uuid.UUID `db:"id"`
	BuyerID       uuid.UUID `db:"buyer_id"`
	ShopID        uuid.UUID `db:"shop_id"`
	Number        string    `db:"number"`
	Status        string    `db:"status"` // pending, paid, processing, shipped, completed, cancelled
	TotalPrice       float64   `db:"total_price"`
	ShippingAddress string  `db:"shipping_address"`
	ShippingCourier  string    `db:"shipping_courier"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type OrderItem struct {
	ID        uuid.UUID `db:"id"`
	OrderID   uuid.UUID `db:"order_id"`
	ItemID    uuid.UUID `db:"item_id"`
	Quantity  int       `db:"quantity"`
	Price     float64   `db:"price"`
	CreatedAt time.Time `db:"created_at"`
}

type OrderItemInput struct {
    ItemID      uuid.UUID `json:"item_id" binding:"required"`
    Quantity    int       `json:"quantity" binding:"required,min=1"`
}

type CreateOrderInput struct {
    Items           []OrderItemInput `json:"items" binding:"required,dive"`
    ShippingAddress string           `json:"shipping_address" binding:"required"`
    ShippingCourier string           `json:"shipping_courier" binding:"required"`
}

// Input untuk FR-BUYER-02: Filter & Pencarian
type ItemFilter struct {
    Keyword     string  `form:"keyword"`
    CategoryID  uuid.UUID `form:"category_id"`
    MinPrice    float64 `form:"min_price"`
    MaxPrice    float64 `form:"max_price"`
    Limit       int     `form:"limit"`
    Offset      int     `form:"offset"`
}

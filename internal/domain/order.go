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
	ShippingAddress string  `db:"shipping_address"`
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


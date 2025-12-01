package entity
import (
	"time"
	"github.com/google/uuid"
)

type Item struct {
	ID          uuid.UUID `db:"id"`
	ShopID      uuid.UUID `db:"shop_id"`
	CategoryID  uuid.UUID `db:"category_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Price       float64   `db:"price"`
	Stock       int       `db:"stock"`
	Condition   string    `db:"condition"`
	Status      string    `db:"status"` // active, inactive, deleted
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type ItemImage struct {
	ID        uuid.UUID `db:"id"`
	ItemID    uuid.UUID `db:"item_id"`
	ImageURL  string    `db:"image_url"`
	CreatedAt time.Time `db:"created_at"`
}

type CreateItemInput struct {
	Name        string  `form:"name" binding:"required"`
	Description string  `form:"description"`
	Price       float64 `form:"price" binding:"required"`
	Stock       int     `form:"stock" binding:"required"`
	Condition   string  `form:"condition" binding:"required"`
	CategoryID  uuid.UUID `form:"category_id" binding:"required"`
}

type UpdateItemInput struct {
    Name        string  `json:"name" binding:"required"`
    Description string  `json:"description"`
    Price       float64 `json:"price" binding:"min=0"`
    Stock       int     `json:"stock" binding:"min=0"` 
    Condition   string  `json:"condition" binding:"required"`
    Status      string  `json:"status"` 
}

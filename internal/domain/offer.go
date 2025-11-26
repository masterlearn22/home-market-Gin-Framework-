package entity

import (
	"time"
	"github.com/google/uuid"
)

type Offer struct {
	ID          uuid.UUID `db:"id"`
	GiverID     uuid.UUID `db:"giver_id"`
	SellerID    uuid.UUID `db:"seller_id"`
	ItemName    string    `db:"item_name"`
	Description string    `db:"description"`
	ImageURL    string    `db:"image_url"`
	ExpectedPrice float64 `db:"expected_price"`
	AgreedPrice   float64 `db:"agreed_price"`
	Status      string    `db:"status"`  // pending, accepted, rejected, paid
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

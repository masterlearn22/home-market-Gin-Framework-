package entity

import (
	"time"
	"github.com/google/uuid"
)

type Shop struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	Name      string    `db:"name"`
	Description string  `db:"description"`
	Address   string    `db:"address"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type CreateShopInput struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Address     string `json:"address" binding:"required"`
}

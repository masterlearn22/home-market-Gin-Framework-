package repository

import (
	"database/sql"
	"errors"
	entity "home-market/internal/domain"
	"github.com/google/uuid"
)

type ShopRepository interface {
	GetByUserID(userID uuid.UUID) (*entity.Shop, error)
	CreateShop(shop *entity.Shop) error
    GetShopOwnerID(shopID uuid.UUID) (uuid.UUID, error) // Dari ItemRepository sebelumnya
    IsCategoryOwnedByShop(categoryID, shopID uuid.UUID) (bool, error)
}

type shopRepository struct {
	db *sql.DB
}

func NewShopRepository(db *sql.DB) ShopRepository {
	return &shopRepository{db: db}
}

func (r *shopRepository) GetByUserID(userID uuid.UUID) (*entity.Shop, error) {
	var shop entity.Shop

	query := `
		SELECT id, user_id, name, description, address, created_at, updated_at
		FROM shops
		WHERE user_id = $1
	`

	err := r.db.QueryRow(query, userID).Scan(
		&shop.ID,
		&shop.UserID,
		&shop.Name,
		&shop.Description,
		&shop.Address,
		&shop.CreatedAt,
		&shop.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // belum punya toko
		}
		return nil, err
	}

	return &shop, nil
}

func (r *shopRepository) CreateShop(shop *entity.Shop) error {
	query := `
		INSERT INTO shops (id, user_id, name, description, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`

	_, err := r.db.Exec(query,
		shop.ID,
		shop.UserID,
		shop.Name,
		shop.Description,
		shop.Address,
	)

	return err
}

// [internal/repository/postgresql/item_repository.go]

func (r *shopRepository) GetShopOwnerID(shopID uuid.UUID) (uuid.UUID, error) {
    var ownerID uuid.UUID
    query := `SELECT user_id FROM shops WHERE id = $1`
    
    err := r.db.QueryRow(query, shopID).Scan(&ownerID)

    if err == sql.ErrNoRows {
        // Kembalikan uuid.Nil jika toko tidak ditemukan
        return uuid.Nil, errors.New("shop not found")
    }
    if err != nil {
        return uuid.Nil, err
    }
    return ownerID, nil
}

func (r *shopRepository) IsCategoryOwnedByShop(categoryID, shopID uuid.UUID) (bool, error) {
	var exists bool

	query := `
		SELECT EXISTS(
			SELECT 1 FROM categories
			WHERE id = $1 AND shop_id = $2
		)
	`

	err := r.db.QueryRow(query, categoryID, shopID).Scan(&exists)
	return exists, err
}
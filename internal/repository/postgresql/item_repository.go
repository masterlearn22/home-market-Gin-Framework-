package repository

import (
	"database/sql"
	entity "home-market/internal/domain"

	"github.com/google/uuid"
)

type ItemRepository interface {
	CreateItem(item *entity.Item) error
	CreateItemImage(img *entity.ItemImage) error
	GetShopByUserID(userID uuid.UUID) (*entity.Shop, error)
	IsCategoryOwnedByShop(categoryID, shopID uuid.UUID) (bool, error)
}

type itemRepository struct {
	db *sql.DB
}

func NewItemRepository(db *sql.DB) ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) GetShopByUserID(userID uuid.UUID) (*entity.Shop, error) {
	var shop entity.Shop

	query := `
		SELECT id, user_id, name, description, address, created_at, updated_at
		FROM shops
		WHERE user_id = $1
	`

	err := r.db.QueryRow(query, userID).Scan(
		&shop.ID, &shop.UserID, &shop.Name, &shop.Description,
		&shop.Address, &shop.CreatedAt, &shop.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &shop, nil
}

func (r *itemRepository) IsCategoryOwnedByShop(categoryID, shopID uuid.UUID) (bool, error) {
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

func (r *itemRepository) CreateItem(item *entity.Item) error {
	query := `
		INSERT INTO items (id, shop_id, category_id, name, description, price, stock, condition, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
	`

	_, err := r.db.Exec(query,
		item.ID, item.ShopID, item.CategoryID, item.Name,
		item.Description, item.Price, item.Stock, item.Condition,
		item.Status,
	)
	return err
}

func (r *itemRepository) CreateItemImage(img *entity.ItemImage) error {
	query := `
		INSERT INTO item_images (id, item_id, image_url, created_at)
		VALUES ($1, $2, $3, NOW())
	`
	_, err := r.db.Exec(query, img.ID, img.ItemID, img.ImageURL)
	return err
}

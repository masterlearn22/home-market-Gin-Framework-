package service

import (
	"errors"
	"time"

	entity "home-market/internal/domain"
	repo "home-market/internal/repository/postgresql"

	"github.com/google/uuid"
)

var (
	ErrInvalidStock      = errors.New("stock must be >= 0")
	ErrInvalidPrice      = errors.New("price must be >= 0")
	ErrCategoryNotOwned  = errors.New("category does not belong to seller's shop")
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

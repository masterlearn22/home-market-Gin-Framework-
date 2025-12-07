package service

import (
	"errors"
	"time"

	entity "home-market/internal/domain"
	repo "home-market/internal/repository/postgresql"
	"github.com/google/uuid"
)

// --- ERROR DEFINITIONS (Consolidated) ---
var (
	// Shop Errors
	ErrNotSeller      = errors.New("only seller role can manage shop/items")
	ErrShopExists     = errors.New("user already has a shop")
	ErrNoShopOwned    = errors.New("seller does not own a shop")
	
	// Category Errors
	ErrCategoryExists = errors.New("category name already exists in this shop")

	// Item Errors
	ErrInvalidStock     = errors.New("stock must be >= 0")
	ErrInvalidPrice     = errors.New("price must be >= 0")
	ErrCategoryNotOwned = errors.New("category does not belong to seller's shop")
)

// --- SERVICE STRUCT (Consolidated Dependencies) ---

type ShopItemService struct {
	// Repositories untuk Shop/Category/Item CRUD
	shopRepo     repo.ShopRepository 
	categoryRepo repo.CategoryRepository
	itemRepo     repo.ItemRepository
	
	// ItemService masih memerlukan OrderRepo untuk GetItemForOrder (Marketplace/Detail)
	orderRepo    repo.OrderRepository 
}

func NewShopItemService(
	shopRepo repo.ShopRepository,
	categoryRepo repo.CategoryRepository,
	itemRepo repo.ItemRepository,
	orderRepo repo.OrderRepository,
) *ShopItemService {
	return &ShopItemService{
		shopRepo:     shopRepo,
		categoryRepo: categoryRepo,
		itemRepo:     itemRepo,
		orderRepo:    orderRepo,
	}
}

// ===============================================
// 1. SHOP METHODS (dari ShopService)
// ===============================================

// @Summary      Create Seller Shop
// @Description  Allows a registered Seller to create their shop. A user can only own one shop.
// @Tags         Shop
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        input body entity.CreateShopInput true "Shop details (Name, Address, Description)"
// @Success      201  {object}  entity.Shop
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{} "Forbidden (not seller)"
// @Failure      409  {object}  map[string]interface{} "Conflict (shop already exists)"
// @Failure      500  {object}  map[string]interface{}
// @Router       /shops [post]
func (s *ShopItemService) CreateShop(userID uuid.UUID, role string, input entity.CreateShopInput) (*entity.Shop, error) {
	if role != "seller" {
		return nil, ErrNotSeller
	}

	existing, err := s.shopRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrShopExists
	}

	shop := &entity.Shop{
		ID: uuid.New(),
		UserID: userID,
		Name: input.Name,
		Address: input.Address,
		Description: input.Description,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.shopRepo.CreateShop(shop); err != nil {
		return nil, err
	}

	return shop, nil
}

// ===============================================
// 2. CATEGORY METHODS (dari CategoryService)
// ===============================================

// @Summary      Create New Shop Category
// @Description  Allows a logged-in seller to create a new category unique to their shop.
// @Tags         Category
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        input body entity.CreateCategoryInput true "Category details (Name)"
// @Success      201  {object}  entity.Category
// @Failure      400  {object}  map[string]interface{} "Invalid input or missing shop"
// @Failure      403  {object}  map[string]interface{} "Forbidden (not seller)"
// @Failure      409  {object}  map[string]interface{} "Conflict (category name already exists)"
// @Failure      500  {object}  map[string]interface{} "Internal server error"
// @Router       /categories [post]
func (s *ShopItemService) CreateCategory(userID uuid.UUID, role string, input entity.CreateCategoryInput) (*entity.Category, error) {

	if role != "seller" {
		return nil, ErrNotSeller
	}

	shop, err := s.shopRepo.GetByUserID(userID) // Menggunakan shopRepo
	if err != nil {
		return nil, err
	}

	if shop == nil {
		return nil, ErrNoShopOwned
	}

	exists, err := s.categoryRepo.ExistsByName(shop.ID, input.Name) // Menggunakan categoryRepo
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCategoryExists
	}

	category := &entity.Category{
		ID: uuid.New(),
		ShopID: shop.ID,
		Name: input.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.categoryRepo.CreateCategory(category) // Menggunakan categoryRepo
	if err != nil {
		return nil, err
	}

	return category, nil
}

// ===============================================
// 3. ITEM CRUD METHODS (dari ItemService)
// ===============================================

// @Summary      Create New Item
// @Description  Allows a Seller to create a new item within their shop. Requires multipart/form-data for input and image upload.
// @Tags         Seller/Items
// @Accept       mpfd
// @Produce      json
// @Security     ApiKeyAuth
// @Param        name formData string true "Item Name"
// @Param        description formData string false "Item Description"
// @Param        price formData number true "Item Price"
// @Param        stock formData integer true "Initial Stock"
// @Param        condition formData string false "Item Condition"
// @Param        category_id formData string true "Category ID (UUID) owned by the shop"
// @Param        images formData file true "Item Images"
// @Success      201  {object}  map[string]interface{} "Returns created item and image URLs"
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{} "Forbidden (not seller)"
// @Failure      500  {object}  map[string]interface{}
// @Router       /items [post]
func (s *ShopItemService) CreateItem(userID uuid.UUID, role string, input entity.CreateItemInput, imageURLs []string) (*entity.Item, []entity.ItemImage, error) {

	if role != "seller" {
		return nil, nil, ErrNotSeller
	}

	shop, err := s.shopRepo.GetByUserID(userID)
	if err != nil {
		return nil, nil, err
	}
	if shop == nil {
		return nil, nil, ErrNoShopOwned
	}

	
	owned, err := s.shopRepo.IsCategoryOwnedByShop(input.CategoryID, shop.ID)
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
		ID: uuid.New(),
		ShopID: shop.ID,
		CategoryID: input.CategoryID,
		Name: input.Name,
		Description: input.Description,
		Price: input.Price,
		Stock: input.Stock,
		Condition: input.Condition,
		Status: "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}


	if err := s.itemRepo.CreateItem(item); err != nil {
		return nil, nil, err
	}


	var images []entity.ItemImage
	for _, url := range imageURLs {
		img := entity.ItemImage{
			ID: uuid.New(),
			ItemID: item.ID,
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

// @Summary      Update Item Details
// @Description  Allows a Seller to update item fields (name, price, stock, status, etc.) for an item they own.
// @Tags         Seller/Items
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Item ID to update"
// @Param        input body entity.UpdateItemInput true "Updated item details"
// @Success      200  {object}  entity.Item "Returns the updated item"
// @Failure      400  {object}  map[string]interface{}
// @Failure      403  {object}  map[string]interface{} "Unauthorized (not owner)"
// @Failure      404  {object}  map[string]interface{} "Item not found"
// @Failure      500  {object}  map[string]interface{}
// @Router       /items/{id} [put]
func (s *ShopItemService) UpdateItem(userID uuid.UUID, itemID uuid.UUID, input entity.UpdateItemInput) (*entity.Item, error) {
	
	item, err := s.itemRepo.GetItemByID(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, errors.New("item not found")
	}

	shop, err := s.shopRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	if shop == nil {
		return nil, errors.New("you do not have a shop")
	}

	if item.ShopID != shop.ID {
		return nil, errors.New("unauthorized: this item does not belong to your shop")
	}


	item.Name = input.Name
	item.Description = input.Description
	item.Price = input.Price
	item.Stock = input.Stock
	item.Condition = input.Condition

	
	if input.Status != "" {
		item.Status = input.Status
	}

	if err := s.itemRepo.UpdateItem(item); err != nil {
		return nil, err
	}

	return item, nil
}

// @Summary      Archive/Delete Item (Soft Delete)
// @Description  Sets the status of an item to 'inactive'. Only the item owner can perform this action.
// @Tags         Seller/Items
// @Accept       json
// @Produce      json
// @Security     ApiKeyAuth
// @Param        id   path      string  true  "Item ID to delete"
// @Success      200  {object}  map[string]interface{} "Item archived/deleted successfully"
// @Failure      403  {object}  map[string]interface{} "Unauthorized"
// @Failure      404  {object}  map[string]interface{} "Item not found"
// @Failure      500  {object}  map[string]interface{}
// @Router       /items/{id} [delete]
func (s *ShopItemService) DeleteItem(userID uuid.UUID, itemID uuid.UUID) error {
	item, err := s.itemRepo.GetItemByID(itemID)
	if err != nil {
		return err
	}
	if item == nil {
		return errors.New("item not found")
	}

	shop, err := s.shopRepo.GetByUserID(userID)
	if err != nil {
		return err
	}
	if shop == nil || item.ShopID != shop.ID {
		return errors.New("unauthorized")
	}

	item.Status = "inactive"
	return s.itemRepo.UpdateItem(item)
}

// @Summary      Get Item Detail (Marketplace View)
// @Description  Retrieves detailed information for a single item, ensuring it is active and available.
// @Tags         Marketplace
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Item ID to retrieve"
// @Success      200  {object}  entity.Item
// @Failure      404  {object}  map[string]interface{} "Item not found or inactive"
// @Failure      500  {object}  map[string]interface{}
// @Router       /market/items/{id} [get]
func (s *ShopItemService) GetItemDetail(itemID uuid.UUID) (*entity.Item, error) {
	// Memerlukan orderRepo untuk GetItemForOrder yang dipakai di Marketplace
	item, err := s.orderRepo.GetItemForOrder(itemID)
	if err != nil {
		return nil, err
	}
	if item == nil || item.Status != "active" {
		return nil, errors.New("item not found or inactive")
	}
	return item, nil
}
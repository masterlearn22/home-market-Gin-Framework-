package route

import (
	"database/sql"
	"log"
	"github.com/google/uuid"
	httpHandler "home-market/internal/delivery/http/handler"
	repo "home-market/internal/repository/postgresql"
	mongorepo "home-market/internal/repository/mongodb"
	service "home-market/internal/service/postgresql"
	"github.com/gin-gonic/gin"
	"home-market/internal/delivery/http/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	swaggerFiles "github.com/swaggo/files"
    ginSwagger "github.com/swaggo/gin-swagger"
	_ "home-market/docs"
)

// Asumsi: NewItemService, NewOfferService, NewOrderService, NewAdminService sudah menerima dependencies yang benar.

func SetupRoute(app *gin.Engine, db *sql.DB, mongoclient *mongo.Client) {
	// --- 1. Ambil default role ---
	var defaultRoleID uuid.UUID
	if err := db.QueryRow(`SELECT id FROM roles WHERE name = $1`, "buyer").Scan(&defaultRoleID); err != nil {
		log.Printf("warning: gagal mengambil default role 'buyer': %v", err)
	}

	// --- 2. INIT REPOSITORIES (Dependencies Inti) ---
	userRepo := repo.NewUserRepository(db)
	shopRepo := repo.NewShopRepository(db)
	categoryRepo := repo.NewCategoryRepository(db)
	itemRepo := repo.NewItemRepository(db)
	orderRepo := repo.NewOrderRepository(db)
	offerRepo := repo.NewOfferRepository(db) // Asumsi OfferRepo ada
	logRepo := mongorepo.NewLogRepository(mongoclient) // MongoDB Log Repo

	// --- 3. INIT SERVICES ---
	authService := service.NewAuthService(userRepo, defaultRoleID)
	shopService := service.NewShopService(shopRepo)
	categoryService := service.NewCategoryService(categoryRepo)
	
    // Service yang membutuhkan banyak dependency:
	itemService := service.NewItemService(itemRepo, shopRepo, orderRepo) // Core Item CRUD
	orderService := service.NewOrderService(orderRepo, shopRepo, logRepo) // Order, Marketplace, Shipment
	offerService := service.NewOfferService(offerRepo, itemRepo, shopRepo, logRepo) // Offer Management
	adminService := service.NewAdminService(userRepo, itemRepo) // Admin Moderation

	// --- 4. INIT HANDLERS ---
	authHandler := httpHandler.NewAuthHandler(authService)
	shopHandler := httpHandler.NewShopHandler(shopService)
	categoryHandler := httpHandler.NewCategoryHandler(categoryService)
	itemHandler := httpHandler.NewItemHandler(itemService) // Item CRUD
	orderHandler := httpHandler.NewOrderHandler(orderService) // Order/Marketplace
	offerHandler := httpHandler.NewOfferHandler(offerService) // Offer Management
	adminHandler := httpHandler.NewAdminHandler(adminService)

	// --- 5. DEFINISIKAN GROUP ROUTE ---
	api := app.Group("/api")

	// --- SWAGGER/OPENAPI DOCUMENTATION ROUTE ---
    app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
        ginSwagger.URL("/swagger/doc.json"),
        ginSwagger.DefaultModelsExpandDepth(0),
    ))

    // --- Authentication & Profile ---
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.GET("/profile", middleware.AuthRequired(), authHandler.Profile)

    // --- Shop & Categories ---
	shop := api.Group("/shops")
	shop.POST("/", middleware.AuthRequired(), shopHandler.CreateShop)
	cat := api.Group("/categories")
	cat.POST("/", middleware.AuthRequired(), categoryHandler.CreateCategory)

    // --- Item CRUD (Seller) ---
	items := api.Group("/items", middleware.AuthRequired())
	items.POST("", itemHandler.CreateItem)
	items.PUT("/:id", itemHandler.UpdateItem)
	items.DELETE("/:id", itemHandler.DeleteItem)

    // --- Offer Management (Giver & Seller) ---
	offers := api.Group("/offers", middleware.AuthRequired())
	offers.POST("", offerHandler.CreateOffer)           // Giver Create Offer
	offers.GET("/my", offerHandler.GetMyOffers)         // Giver View Offers
	offers.GET("/inbox", offerHandler.GetOffersToSeller) // Seller View Inbox
	offers.POST("/:id/accept", offerHandler.AcceptOffer)
	offers.POST("/:id/reject", offerHandler.RejectOffer)

    // --- Marketplace (Public/Buyer) ---
	market := api.Group("/market")
	market.GET("/items", orderHandler.GetMarketplaceItems) // Buyer/Public View
	market.GET("/items/:id", orderHandler.GetItemDetail) // Buyer/Public Detail

    // --- Orders & Shipment ---
	orders := api.Group("/orders")
	orders.POST("", middleware.AuthRequired(), orderHandler.CreateOrder) // Buyer Create Order
	
    // Seller/Admin Management
	orders.PATCH("/:id/status", middleware.AuthRequired(), orderHandler.UpdateOrderStatus)
	orders.POST("/:id/shipping", middleware.AuthRequired(), orderHandler.InputShippingReceipt)
	
    // Buyer/Admin Tracking
	orders.GET("/:id/tracking", middleware.AuthRequired(), orderHandler.GetOrderTracking)


    // --- Admin Group ---
	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequired(), middleware.RoleAllowed("admin")) 
	
	admin.GET("/users", adminHandler.ListUsers)
	admin.PATCH("/users/:id/status", adminHandler.BlockUser) 
	admin.PATCH("/items/:id/moderate", adminHandler.ModerateItem)
}
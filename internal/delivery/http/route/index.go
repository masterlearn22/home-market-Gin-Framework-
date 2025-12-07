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
	offerRepo := repo.NewOfferRepository(db) 
	logRepo := mongorepo.NewLogRepository(mongoclient) 

	// --- 3. INIT SERVICES ---
	authService := service.NewAuthService(userRepo, defaultRoleID)
	
	// INIT SERVICE GABUNGAN (Shop, Category, Item CRUD)
	shopItemService := service.NewShopItemService(shopRepo, categoryRepo, itemRepo, orderRepo) 

	// Service yang tetap terpisah
	orderService := service.NewOrderService(orderRepo, shopRepo, logRepo) 
	offerService := service.NewOfferService(offerRepo, itemRepo, shopRepo, logRepo) 
	adminService := service.NewAdminService(userRepo, itemRepo) 

	// --- 4. INIT HANDLERS ---
	authHandler := httpHandler.NewAuthHandler(authService)
	// INIT HANDLER GABUNGAN
	shopItemHandler := httpHandler.NewShopItemHandler(shopItemService) 

	// Handlers yang tetap terpisah
	orderHandler := httpHandler.NewOrderHandler(orderService) 
	offerHandler := httpHandler.NewOfferHandler(offerService) 
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

	// --- Shop & Categories (Arahkan ke Handler Gabungan) ---
	shop := api.Group("/shops")
	shop.POST("/", middleware.AuthRequired(), shopItemHandler.CreateShop) // DIGANTI
	cat := api.Group("/categories")
	cat.POST("/", middleware.AuthRequired(), shopItemHandler.CreateCategory) // DIGANTI

	// --- Item CRUD (Seller) (Arahkan ke Handler Gabungan) ---
	items := api.Group("/items", middleware.AuthRequired())
	items.POST("", shopItemHandler.CreateItem) // DIGANTI
	items.PUT("/:id", shopItemHandler.UpdateItem) // DIGANTI
	items.DELETE("/:id", shopItemHandler.DeleteItem) // DIGANTI

	// --- Offer Management (Giver & Seller) (TIDAK BERUBAH) ---
	offers := api.Group("/offers", middleware.AuthRequired())
	offers.POST("", offerHandler.CreateOffer) 
	offers.GET("/my", offerHandler.GetMyOffers) 
	offers.GET("/inbox", offerHandler.GetOffersToSeller) 
	offers.POST("/:id/accept", offerHandler.AcceptOffer)
	offers.POST("/:id/reject", offerHandler.RejectOffer)

	// --- Marketplace & Orders (TIDAK BERUBAH) ---
	market := api.Group("/market")
	market.GET("/items", orderHandler.GetMarketplaceItems) 
	market.GET("/items/:id", orderHandler.GetItemDetail) 

	orders := api.Group("/orders")
	orders.POST("", middleware.AuthRequired(), orderHandler.CreateOrder) 
	
	orders.PATCH("/:id/status", middleware.AuthRequired(), orderHandler.UpdateOrderStatus)
	orders.POST("/:id/shipping", middleware.AuthRequired(), orderHandler.InputShippingReceipt)
	
	orders.GET("/:id/tracking", middleware.AuthRequired(), orderHandler.GetOrderTracking)


	// --- Admin Group (TIDAK BERUBAH) ---
	admin := api.Group("/admin")
	admin.Use(middleware.AuthRequired(), middleware.RoleAllowed("admin")) 
	
	admin.GET("/users", adminHandler.ListUsers)
	admin.PATCH("/users/:id/status", adminHandler.BlockUser) 
	admin.PATCH("/items/:id/moderate", adminHandler.ModerateItem)
}
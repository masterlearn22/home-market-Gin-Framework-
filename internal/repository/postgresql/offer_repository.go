package repository

import(
	"database/sql"
	entity "home-market/internal/domain"
	"github.com/google/uuid"
)
type offerRepository struct {
	db *sql.DB
}

type OfferRepository interface {
    CreateOffer(offer *entity.Offer) error
    GetOffersByGiverID(giverID uuid.UUID) ([]entity.Offer, error)
    GetOffersBySellerID(sellerID uuid.UUID) ([]entity.Offer, error)
    GetOfferByID(offerID uuid.UUID) (*entity.Offer, error)
    UpdateOffer(offer *entity.Offer) error
}

func NewOfferRepository(db *sql.DB) OfferRepository {
	return &offerRepository{db: db}
}

func (r *offerRepository) CreateOffer(offer *entity.Offer) error {
    query := `
        INSERT INTO offers (id, giver_id, seller_id, item_name, description, image_url, expected_price, condition, location, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
    `
    // seller_id harus diubah ke interface{} atau sql.NullUUID jika boleh NULL
    _, err := r.db.Exec(query,
        offer.ID, offer.GiverID, offer.SellerID, offer.ItemName, offer.Description,
        offer.ImageURL, offer.ExpectedPrice, offer.Condition, offer.Location, offer.Status,
    )
    return err
}

// FR-GIVER-03: Melihat Status Penawaran
func (r *offerRepository) GetOffersByGiverID(giverID uuid.UUID) ([]entity.Offer, error) {
    var offers []entity.Offer
    query := `
        SELECT id, giver_id, seller_id, item_name, description, image_url, expected_price, agreed_price, condition, location, status, created_at, updated_at
        FROM offers
        WHERE giver_id = $1
    `
    rows, err := r.db.Query(query, giverID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        var offer entity.Offer
        err := rows.Scan(
            &offer.ID, &offer.GiverID, &offer.SellerID, &offer.ItemName, &offer.Description,
            &offer.ImageURL, &offer.ExpectedPrice, &offer.AgreedPrice, &offer.Condition, &offer.Location, &offer.Status, &offer.CreatedAt, &offer.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        offers = append(offers, offer)
    }
    return offers, nil
}

// FR-OFFER-01: Seller Melihat Penawaran
func (r *offerRepository) GetOffersBySellerID(sellerID uuid.UUID) ([]entity.Offer, error) {
    var offers []entity.Offer
    query := `
        SELECT id, giver_id, seller_id, item_name, description, image_url, expected_price, agreed_price, condition, location, status, created_at, updated_at
        FROM offers
        WHERE seller_id = $1 OR seller_id IS NULL -- Jika open offer juga diizinkan dilihat, sesuaikan query ini
    `
    rows, err := r.db.Query(query, sellerID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        // Asumsi struct Offer sudah menggunakan sql.NullFloat64 untuk agreed_price
        var offer entity.Offer
        err := rows.Scan(
            &offer.ID, &offer.GiverID, &offer.SellerID, &offer.ItemName, &offer.Description,
            &offer.ImageURL, &offer.ExpectedPrice, &offer.AgreedPrice, &offer.Condition, &offer.Location, &offer.Status, &offer.CreatedAt, &offer.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        offers = append(offers, offer)
    }
    return offers, nil
}

// Diperlukan untuk mengecek ownership sebelum update status
func (r *offerRepository) GetOfferByID(offerID uuid.UUID) (*entity.Offer, error) {
    var offer entity.Offer
    query := `
        SELECT id, giver_id, seller_id, item_name, description, image_url, expected_price, agreed_price, condition, location, status, created_at, updated_at
        FROM offers WHERE id = $1
    `
    // Asumsi struct Offer sudah menggunakan sql.NullFloat64
    err := r.db.QueryRow(query, offerID).Scan(
        &offer.ID, &offer.GiverID, &offer.SellerID, &offer.ItemName, &offer.Description,
        &offer.ImageURL, &offer.ExpectedPrice, &offer.AgreedPrice, &offer.Condition, &offer.Location, &offer.Status, &offer.CreatedAt, &offer.UpdatedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    return &offer, err
}

// FR-OFFER-02/03: Update Status dan Agreed Price
func (r *offerRepository) UpdateOffer(offer *entity.Offer) error {
    query := `
        UPDATE offers
        SET status=$1, agreed_price=$2, updated_at=NOW()
        WHERE id=$3
    `
    // agreed_price (offer.AgreedPrice) sekarang bertipe sql.NullFloat64
    _, err := r.db.Exec(query, offer.Status, offer.AgreedPrice, offer.ID)
    return err
}
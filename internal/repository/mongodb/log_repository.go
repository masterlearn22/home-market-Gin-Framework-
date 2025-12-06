package mongodb

import (
	"context"
	"fmt"
	"time"
	// Asumsi Anda mengimpor entity dari domain
	entity "home-market/internal/domain"

	"go.mongodb.org/mongo-driver/mongo"
)

// PLACEHOLDERS (Anda perlu menentukan nama Database dan Collection di sini)
const (
	// Ganti dengan nama database MongoDB Anda
	DatabaseName = "random_home_market" 
	
	// Collection untuk menyimpan riwayat status (FR-ORDER-02/03)
	CollectionStatus = "history_status"
)

// LogRepository Interface (sudah ada)
type LogRepository interface {
	SaveHistoryStatus(doc *entity.HistoryStatus) error
}

type logRepository struct {
	collection *mongo.Collection
}

// NewLogRepository: Constructor untuk inisialisasi LogRepository
func NewLogRepository(client *mongo.Client) LogRepository {
	db := client.Database(DatabaseName)
	return &logRepository{
		collection: db.Collection(CollectionStatus),
	}
}

// FR-ORDER-02/03: Simpan Riwayat Status
func (r *logRepository) SaveHistoryStatus(doc *entity.HistoryStatus) error {
	// Set timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Implementasi Insert ke MongoDB Collection 'history_status'
	// Driver Mongo akan secara otomatis mengisi _id (ObjectId) jika belum diset.
	
	_, err := r.collection.InsertOne(ctx, doc)
	
	if err != nil {
		// Gunakan fmt.Errorf untuk membungkus error agar mudah di-debug
		return fmt.Errorf("failed to insert history status to Mongo: %w", err)
	}

	return nil
}
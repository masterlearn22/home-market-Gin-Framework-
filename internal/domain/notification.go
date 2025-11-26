package entity
import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Notification struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    string             `bson:"user_id"`
	Type      string             `bson:"type"`      // offer_created, order_status, etc
	Title     string             `bson:"title"`
	Message   string             `bson:"message"`
	CreatedAt time.Time          `bson:"created_at"`
}

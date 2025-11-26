package entity
import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ActivityLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    string             `bson:"user_id"`
	Action    string             `bson:"action"`
	Module    string             `bson:"module"`
	IPAddress string             `bson:"ip_address"`
	Device    string             `bson:"device"`
	CreatedAt time.Time          `bson:"created_at"`
}

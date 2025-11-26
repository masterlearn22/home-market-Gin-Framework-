package entity
import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HistoryStatus struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	RelatedID  string             `bson:"related_id"` // item/order/offer id
	OldStatus  string             `bson:"old_status"`
	NewStatus  string             `bson:"new_status"`
	UpdatedBy  string             `bson:"updated_by"`
	Note       string             `bson:"note"`
	Timestamp  time.Time          `bson:"timestamp"`
}


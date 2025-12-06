package entity
import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type HistoryStatus struct {
    ID           primitive.ObjectID `bson:"_id" json:"id"`
    RelatedID    string             `bson:"related_id" json:"relatedId"`
    RelatedType  string             `bson:"related_type" json:"relatedType"`
    OldStatus    string             `bson:"old_status" json:"oldStatus"`
    NewStatus    string             `bson:"new_status" json:"newStatus"`
    ChangedBy    string             `bson:"changed_by" json:"changedBy"`      
    Timestamp    time.Time          `bson:"timestamp" json:"timestamp"`
    Note         string             `bson:"note" json:"note"`
}


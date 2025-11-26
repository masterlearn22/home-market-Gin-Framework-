package entity
import (
	"github.com/google/uuid"
)

type UserRole struct {
	UserID uuid.UUID `db:"user_id"`
	RoleID uuid.UUID `db:"role_id"`
}

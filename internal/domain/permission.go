package entity
import (
	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`         
	Resource    string    `json:"resource" db:"resource"` 
	Action      string    `json:"action" db:"action"`     
	Description string    `json:"description" db:"description"`
}

type RolePermission struct {
	RoleID       uuid.UUID `json:"roleId" db:"role_id"`
	PermissionID uuid.UUID `json:"permissionId" db:"permission_id"`
}

package models

import "time"

// UserRole 用户角色表
type UserRole struct {
	ID        int       `gorm:"primaryKey" json:"id"`
	UserID    int       `gorm:"column:user_id" json:"user_id"`
	Role      string    `gorm:"column:role" json:"role"` // admin / user
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (UserRole) TableName() string {
	return "user_roles"
}

// Permission 权限表
type Permission struct {
	ID           int       `gorm:"primaryKey" json:"id"`
	UserID       int       `gorm:"column:user_id" json:"user_id"`
	ResourceType string    `gorm:"column:resource_type" json:"resource_type"` // rule / node
	ResourceID   int       `gorm:"column:resource_id" json:"resource_id"`
	Action       string    `gorm:"column:action" json:"action"` // read / write
	CreatedAt    time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Permission) TableName() string {
	return "permissions"
}

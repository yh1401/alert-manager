package models

import "time"

// AuditLog 审计日志表 - 记录所有关键操作
type AuditLog struct {
	ID           int       `gorm:"primaryKey" json:"id"`
	UserID       int       `gorm:"column:user_id;index" json:"user_id"`             // 操作人ID
	Username     string    `gorm:"column:username" json:"username"`                 // 操作人用户名（冗余，方便查询）
	ResourceType string    `gorm:"column:resource_type;index" json:"resource_type"` // 资源类型: rule / node
	ResourceID   int       `gorm:"column:resource_id;index" json:"resource_id"`     // 资源ID
	ResourceName string    `gorm:"column:resource_name" json:"resource_name"`       // 资源名称（冗余）
	Action       string    `gorm:"column:action;index" json:"action"`               // 操作类型: create / update / delete / rollback
	OldValue     string    `gorm:"column:old_value;type:text" json:"old_value"`     // 操作前的值（JSON格式）
	NewValue     string    `gorm:"column:new_value;type:text" json:"new_value"`     // 操作后的值（JSON格式）
	Description  string    `gorm:"column:description" json:"description"`           // 操作描述
	IPAddress    string    `gorm:"column:ip_address" json:"ip_address"`             // 操作来源IP
	CreatedAt    time.Time `gorm:"column:created_at;index" json:"created_at"`       // 操作时间
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

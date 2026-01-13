package models

import "time"

// RuleGroup 对应数据库中的 rule_groups 表
type RuleGroup struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	NodeID      int       `gorm:"column:node_id;index:idx_node_file_path,unique" json:"node_id"`
	FilePath    string    `gorm:"column:file_path;index:idx_node_file_path,unique" json:"file_path"`
	Name        string    `gorm:"column:name" json:"name"`
	FileContent string    `gorm:"column:file_content" json:"file_content"`
	IsActive    bool      `gorm:"column:is_active" json:"is_active"`
	Version     int       `gorm:"column:version" json:"version"` // 当前版本号
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (RuleGroup) TableName() string {
	return "rule_groups"
}

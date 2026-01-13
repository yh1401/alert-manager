package models

import "time"

// RuleGroupVersion 存储规则的历史版本
type RuleGroupVersion struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	RuleGroupID int       `gorm:"column:rule_group_id" json:"rule_group_id"` // 关联主表 ID
	NodeID      int       `gorm:"column:node_id" json:"node_id"`             // [快照] 当时的节点
	FilePath    string    `gorm:"column:file_path" json:"file_path"`         // [快照] 目标文件路径
	Name        string    `gorm:"column:name" json:"name"`                   // [快照] 当时的规则名
	FileContent string    `gorm:"column:file_content" json:"file_content"`   // 当时的规则内容
	Version     int       `gorm:"column:version" json:"version"`             // 版本号
	Comment     string    `gorm:"column:comment" json:"comment"`             // 变更说明 (如: "修改阈值为 80%")
	CreatedAt   time.Time `gorm:"column:created_at" json:"created_at"`
	CreatedBy   string    `gorm:"column:created_by" json:"created_by"` // 操作人 (预留)
}

func (RuleGroupVersion) TableName() string {
	return "rule_group_versions"
}

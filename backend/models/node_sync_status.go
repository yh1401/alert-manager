package models

import "time"

// NodeSyncStatus 保存 Agent 最新一次配置同步与重载结果
// 每个节点一行，使用 node_id 唯一索引便于覆盖更新
// fetch_status: updated | not_modified | unchanged | failed
// reload_status: success | skipped | failed
// error_msg: 失败原因（可选）
type NodeSyncStatus struct {
	ID           int       `gorm:"primaryKey"`
	NodeID       int       `gorm:"uniqueIndex"`
	ConfigHash   string    `gorm:"column:config_hash"`
	FetchStatus  string    `gorm:"column:fetch_status"`
	ReloadStatus string    `gorm:"column:reload_status"`
	ErrorMsg     string    `gorm:"column:error_msg"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (NodeSyncStatus) TableName() string {
	return "node_sync_statuses"
}

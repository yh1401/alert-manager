package models

import "time"

// NodeSyncHistory 持久化记录 Agent 每次拉取与重载结果的历史
// fetch_status: updated | not_modified | unchanged | failed
// reload_status: success | skipped | failed
// error_msg: 失败原因（可选）
type NodeSyncHistory struct {
	ID           int       `gorm:"primaryKey"`
	NodeID       int       `gorm:"index;column:node_id"` // 关联节点
	ConfigHash   string    `gorm:"column:config_hash"`   // 配置哈希
	FetchStatus  string    `gorm:"column:fetch_status"`  // 拉取状态
	ReloadStatus string    `gorm:"column:reload_status"` // 重载状态
	ErrorMsg     string    `gorm:"column:error_msg"`     // 错误信息
	CreatedAt    time.Time `gorm:"column:created_at"`    // 创建时间
}

func (NodeSyncHistory) TableName() string {
	return "node_sync_histories"
}

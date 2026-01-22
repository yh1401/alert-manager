package handlers

import (
	"alert-manager-backend/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 辅助函数：字符串转整数
func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	return i
}

type AgentHandler struct {
	BaseHandler
}

func NewAgentHandler() *AgentHandler {
	return &AgentHandler{}
}

// ServeAgentBinary 提供 agent 二进制文件下载
// 路径（相对于后端可执行程序的工作目录）：static/agent/agent
// 前端可以调用此 handler 来触发下载，例如 GET /api/agent/download
func (h *AgentHandler) ServeAgentBinary(c *gin.Context) {
	// 文件路径：在仓库中我们把二进制放在 backend/static/agent/agent
	// 生产部署时请确保可执行程序的工作目录包含该 static 目录，或改为绝对路径
	filePath := "static/agent/agent"

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent binary not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to access file"})
		return
	}

	// 若为目录或大小为0，视为不可用
	if info.IsDir() || info.Size() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid agent binary"})
		return
	}

	// 设置缓存头（可选，根据需要调整）
	c.Header("Cache-Control", "private, max-age=3600")
	// 强制下载并指定下载文件名为 agent（如果需要保留扩展名，可改为 agent.exe）
	c.Header("Content-Disposition", "attachment; filename=agent")

	// 使用 gin 提供的 File 方法发送文件内容
	c.File(filePath)
}

// helper: get username by userID
func (h *AgentHandler) getUsername(userID int) string {
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return "unknown"
	}
	return user.Username
}

// helper: create audit log for node operations
func (h *AgentHandler) createNodeAuditLog(c *gin.Context, userID int, resourceID int, resourceName string, action string, oldValue interface{}, newValue interface{}, description string) {
	username := h.getUsername(userID)
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	log := models.AuditLog{
		UserID:       userID,
		Username:     username,
		ResourceType: "node",
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Action:       action,
		OldValue:     string(oldJSON),
		NewValue:     string(newJSON),
		Description:  description,
		IPAddress:    c.ClientIP(),
		CreatedAt:    time.Now(),
	}
	h.DB.Create(&log)
}

// helper: create audit log for rule operations
func (h *AgentHandler) createRuleAuditLog(c *gin.Context, userID int, resourceID int, resourceName string, action string, oldValue interface{}, newValue interface{}, description string) {
	username := h.getUsername(userID)
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	log := models.AuditLog{
		UserID:       userID,
		Username:     username,
		ResourceType: "rule",
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Action:       action,
		OldValue:     string(oldJSON),
		NewValue:     string(newJSON),
		Description:  description,
		IPAddress:    c.ClientIP(),
		CreatedAt:    time.Now(),
	}
	h.DB.Create(&log)
}

// ReportSyncStatus 记录 Agent 拉取与重载结果
func (h *AgentHandler) ReportSyncStatus(c *gin.Context) {
	var req struct {
		NodeID       int    `json:"node_id"`
		ConfigHash   string `json:"config_hash"`
		FetchStatus  string `json:"fetch_status"`
		ReloadStatus string `json:"reload_status"`
		ErrorMsg     string `json:"error_msg"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.NodeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing node_id"})
		return
	}

	status := models.NodeSyncStatus{
		NodeID:       req.NodeID,
		ConfigHash:   req.ConfigHash,
		FetchStatus:  req.FetchStatus,
		ReloadStatus: req.ReloadStatus,
		ErrorMsg:     req.ErrorMsg,
		UpdatedAt:    time.Now(),
	}

	// upsert by node_id
	if err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_hash", "fetch_status", "reload_status", "error_msg", "updated_at"}),
	}).Create(&status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error: " + err.Error()})
		return
	}

	// persist history when not a no-op
	shouldRecord := !(status.FetchStatus == "not_modified" || status.FetchStatus == "unchanged") || status.ReloadStatus != "skipped"
	if shouldRecord {
		history := models.NodeSyncHistory{
			NodeID:       status.NodeID,
			ConfigHash:   status.ConfigHash,
			FetchStatus:  status.FetchStatus,
			ReloadStatus: status.ReloadStatus,
			ErrorMsg:     status.ErrorMsg,
			CreatedAt:    time.Now(),
		}
		if err := h.DB.Create(&history).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error: " + err.Error()})
			return
		}

		// 写入审计日志
		var node models.Node
		if res := h.DB.First(&node, status.NodeID); res.Error == nil {
			desc := fmt.Sprintf("同步报告: 拉取=%s, 重载=%s", status.FetchStatus, status.ReloadStatus)
			if status.ReloadStatus == "failed" || status.FetchStatus == "failed" {
				desc += fmt.Sprintf(" | 错误: %s", status.ErrorMsg)
			}
			h.createNodeAuditLog(c, 0, status.NodeID, node.Name, "report_sync", nil, status, desc)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ExportConfig 供 Agent 拉取配置
func (h *AgentHandler) ExportConfig(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		c.JSON(400, gin.H{"error": "missing node_id"})
		return
	}

	var node models.Node
	if err := h.DB.Where("id = ?", nodeID).First(&node).Error; err != nil {
		c.JSON(404, gin.H{"error": "node not found"})
		return
	}

	var rules []models.RuleGroup
	if err := h.DB.Where("node_id = ? AND is_active = ?", node.ID, true).Order("file_path asc").Find(&rules).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error: " + err.Error()})
		return
	}

	type filePayload struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	payload := make([]filePayload, 0, len(rules))
	for _, r := range rules {
		payload = append(payload, filePayload{FilePath: r.FilePath, Content: r.FileContent})
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(500, gin.H{"error": "encode error"})
		return
	}

	sum := sha256.Sum256(jsonBytes)
	hash := hex.EncodeToString(sum[:])
	etag := fmt.Sprintf("W/\"%s\"", hash)

	if inm := c.GetHeader("If-None-Match"); inm != "" {
		// 兼容弱标签 W/"hash" 与强标签 "hash"
		normalized := strings.TrimPrefix(strings.TrimSpace(inm), "W/")
		normalized = strings.Trim(normalized, "\"")
		if normalized == hash {
			c.Header("ETag", etag)
			c.Header("X-Config-Hash", hash)
			c.Header("Cache-Control", "public, max-age=30")
			c.Status(304)
			return
		}
	}

	c.Header("ETag", etag)
	c.Header("X-Config-Hash", hash)
	c.Header("Cache-Control", "no-cache")
	c.Data(200, "application/json", jsonBytes)
}

// UpdateHeartbeat 接收 Agent 心跳
func (h *AgentHandler) UpdateHeartbeat(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		c.JSON(400, gin.H{"error": "missing node_id"})
		return
	}

	updates := map[string]interface{}{
		"last_heartbeat": time.Now(),
	}

	// 动态更新 IP
	if ip := c.Query("ip_address"); ip != "" {
		updates["ip_address"] = ip
	}

	// 使用 Updates 支持多字段更新
	if err := h.DB.Model(&models.Node{}).Where("id = ?", nodeID).Updates(updates).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to update heartbeat"})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

// RegisterNode 处理 Agent 自动注册
func (h *AgentHandler) RegisterNode(c *gin.Context) {
	type filePayload struct {
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	var req struct {
		Hostname  string        `json:"hostname"`
		IPAddress string        `json:"ip_address"`
		Files     []filePayload `json:"files"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	// 创建新节点
	newNode := models.Node{
		Name:          req.Hostname,
		IPAddress:     req.IPAddress,
		LastHeartbeat: time.Now(),
	}

	if err := h.DB.Create(&newNode).Error; err != nil {
		c.JSON(500, gin.H{"error": "failed to create node"})
		return
	}

	// 记录审计日志（系统自动注册，userID 为 0）
	h.createNodeAuditLog(c, 0, newNode.ID, newNode.Name, "create", nil, map[string]interface{}{
		"name":        newNode.Name,
		"ip_address":  newNode.IPAddress,
		"files_count": len(req.Files),
	}, fmt.Sprintf("节点自动注册: %s (%s)", newNode.Name, newNode.IPAddress))

	// 处理随节点注册上报的初始规则文件
	for _, f := range req.Files {
		if strings.TrimSpace(f.Content) == "" {
			continue
		}

		// 简单的名称生成策略：取文件名
		name := filepath.Base(f.FilePath)
		if name == "." || name == "/" {
			name = "imported-rule"
		}

		rule := models.RuleGroup{
			NodeID:      newNode.ID,
			FilePath:    f.FilePath,
			Name:        name,
			FileContent: f.Content,
			IsActive:    true,
			Version:     1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Comment:     "随节点注册一齐注册（Auto-imported during node registration）",
		}

		if err := h.DB.Create(&rule).Error; err != nil {
			fmt.Printf("Error creating initial rule for node %d: %v\n", newNode.ID, err)
			continue
		}

		// 记录规则创建审计日志
		h.createRuleAuditLog(c, 0, rule.ID, rule.Name, "create", nil, map[string]interface{}{
			"node_id":   rule.NodeID,
			"file_path": rule.FilePath,
			"version":   1,
		}, fmt.Sprintf("自动导入规则: %s (节点注册)", rule.Name))
	}

	c.JSON(200, gin.H{"node_id": newNode.ID, "message": "registered successfully"})
}

// ListNodes 列出所有节点与心跳状态
func (h *AgentHandler) ListNodes(c *gin.Context) {
	// auth: require JWT (middleware applied in routes); fetch user
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)
	// 可选：离线阈值（秒），默认 180 秒
	offlineSec := 180
	if v := c.Query("offline_sec"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			offlineSec = n
		}
	}

	var nodes []models.Node
	// admin: see all
	if h.isAdmin(uid) {
		if err := h.DB.Preload("Tags").Order("updated_at desc").Find(&nodes).Error; err != nil {
			c.JSON(500, gin.H{"error": "db error: " + err.Error()})
			return
		}
	} else {
		// non-admin: filter by permissions on nodes (read or write)
		type Row struct{ ResourceID int }
		var rows []Row
		if err := h.DB.Model(&models.Permission{}).
			Select("resource_id").
			Where("user_id = ? AND resource_type = ? AND action IN ?", uid, "node", []string{"read", "write"}).
			Find(&rows).Error; err != nil {
			c.JSON(500, gin.H{"error": "db error: " + err.Error()})
			return
		}
		if len(rows) == 0 {
			c.JSON(200, gin.H{"data": []gin.H{}, "offline_sec": offlineSec})
			return
		}
		ids := make([]int, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ResourceID)
		}
		if err := h.DB.Preload("Tags").Where("id IN ?", ids).Order("updated_at desc").Find(&nodes).Error; err != nil {
			c.JSON(500, gin.H{"error": "db error: " + err.Error()})
			return
		}
	}

	// ----
	// 获取最近一次有意义的同步信息
	historyMap := make(map[int]*models.NodeSyncHistory)
	if len(nodes) > 0 {
		nodeIDs := make([]int, len(nodes))
		for i, n := range nodes {
			nodeIDs[i] = n.ID
		}

		boringStatuses := []string{"not_modified", "unchanged", "skipped"}
		var histories []models.NodeSyncHistory

		err := h.DB.Raw(`
            SELECT * FROM (
                SELECT *, ROW_NUMBER() OVER(PARTITION BY node_id ORDER BY created_at DESC) as rn
                FROM node_sync_histories
                WHERE node_id IN (?)
                AND (fetch_status NOT IN ? OR reload_status NOT IN ?)
            ) t WHERE rn = 1
        `, nodeIDs, boringStatuses, boringStatuses).Scan(&histories).Error

		if err != nil {
			c.JSON(500, gin.H{"error": "db error while fetching histories: " + err.Error()})
			return
		}

		for i := range histories {
			historyMap[histories[i].NodeID] = &histories[i]
		}
	}
	// ----

	// 计算状态
	now := time.Now()

	type NodeInfo struct {
		ID                        int                     `json:"id"`
		Name                      string                  `json:"name"`
		IPAddress                 string                  `json:"ip_address"`
		LastHeartbeat             time.Time               `json:"last_heartbeat"`
		CreatedAt                 time.Time               `json:"created_at"`
		UpdatedAt                 time.Time               `json:"updated_at"`
		Status                    string                  `json:"status"`
		Tags                      []*models.Tag           `json:"tags,omitempty"`
		LastMeaningfulSyncHistory *models.NodeSyncHistory `json:"last_meaningful_sync_history,omitempty"`
	}

	resp := make([]NodeInfo, 0, len(nodes))
	for _, n := range nodes {
		status := "offline"
		if !n.LastHeartbeat.IsZero() && now.Sub(n.LastHeartbeat).Seconds() < float64(offlineSec) {
			status = "online"
		}
		resp = append(resp, NodeInfo{
			ID:                        n.ID,
			Name:                      n.Name,
			IPAddress:                 n.IPAddress,
			LastHeartbeat:             n.LastHeartbeat,
			CreatedAt:                 n.CreatedAt,
			UpdatedAt:                 n.UpdatedAt,
			Status:                    status,
			Tags:                      n.Tags,
			LastMeaningfulSyncHistory: historyMap[n.ID],
		})
	}

	c.JSON(200, gin.H{"data": resp, "offline_sec": offlineSec})
}

// GetNodeDetail 获取单个节点详情
func (h *AgentHandler) GetNodeDetail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing id"})
		return
	}

	// permission: read or admin
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "not authenticated"})
		return
	}

	var node models.Node
	if err := h.DB.Preload("Tags").Where("id = ?", id).First(&node).Error; err != nil {
		c.JSON(404, gin.H{"error": "node not found"})
		return
	}

	if !h.hasNodePermission(uidVal.(int), int(node.ID), "read") {
		c.JSON(403, gin.H{"error": "no read permission"})
		return
	}

	// 计算在线状态（使用默认 180s 或者传入 offline_sec）
	offlineSec := 180
	if v := c.Query("offline_sec"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			offlineSec = n
		}
	}
	status := "offline"
	if !node.LastHeartbeat.IsZero() && time.Since(node.LastHeartbeat).Seconds() < float64(offlineSec) {
		status = "online"
	}

	// latest sync/reload status (optional)
	var sync models.NodeSyncStatus
	var syncData interface{}
	if err := h.DB.Where("node_id = ?", node.ID).First(&sync).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(500, gin.H{"error": "db error: " + err.Error()})
			return
		}
	} else {
		syncData = gin.H{
			"config_hash":   sync.ConfigHash,
			"fetch_status":  sync.FetchStatus,
			"reload_status": sync.ReloadStatus,
			"error_msg":     sync.ErrorMsg,
			"updated_at":    sync.UpdatedAt,
		}
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"id":             node.ID,
			"name":           node.Name,
			"ip_address":     node.IPAddress,
			"last_heartbeat": node.LastHeartbeat,
			"created_at":     node.CreatedAt,
			"updated_at":     node.UpdatedAt,
			"status":         status,
			"sync_status":    syncData,
			"tags":           node.Tags,
		},
		"offline_sec": offlineSec,
	})
}

// ListNodeSyncHistory 获取指定节点的同步/重载历史（按创建时间倒序，默认 50 条）
// 需要节点读权限
func (h *AgentHandler) ListNodeSyncHistory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing id"})
		return
	}

	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "not authenticated"})
		return
	}

	var node models.Node
	if err := h.DB.Where("id = ?", id).First(&node).Error; err != nil {
		c.JSON(404, gin.H{"error": "node not found"})
		return
	}
	if !h.hasNodePermission(uidVal.(int), int(node.ID), "read") {
		c.JSON(403, gin.H{"error": "no read permission"})
		return
	}

	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var histories []models.NodeSyncHistory
	if err := h.DB.Where("node_id = ?", node.ID).Order("created_at desc").Limit(limit).Find(&histories).Error; err != nil {
		c.JSON(500, gin.H{"error": "db error: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": histories, "limit": limit})
}

// ManualSync 手动触发节点的拉取与重载（由前端按钮调用）
// 实际执行仍由 Agent 轮询/心跳触发；此处主要做权限校验与审计记录
func (h *AgentHandler) ManualSync(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing id"})
		return
	}

	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "not authenticated"})
		return
	}

	var node models.Node
	if err := h.DB.Where("id = ?", id).First(&node).Error; err != nil {
		c.JSON(404, gin.H{"error": "node not found"})
		return
	}

	uid := uidVal.(int)
	if !h.hasNodePermission(uid, int(node.ID), "write") {
		c.JSON(403, gin.H{"error": "no write permission"})
		return
	}

	// 记录审计日志
	h.createNodeAuditLog(c, uid, int(node.ID), node.Name, "manual_sync", nil, nil,
		fmt.Sprintf("手动触发节点同步与重载: %s", node.Name))

	c.JSON(http.StatusOK, gin.H{
		"status":  "accepted",
		"message": "已触发手动同步，Agent 将在下次轮询/心跳时拉取与重载",
	})
}

// DeleteNode 删除节点并将该节点下的规则置为失效（需要写权限或管理员）
func (h *AgentHandler) DeleteNode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing id"})
		return
	}

	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)

	// 权限：管理员可以删除，非管理员需要对该节点有 write 权限
	if !h.isAdmin(uid) {
		if !h.hasNodePermission(uid, parseInt(id), "write") {
			c.JSON(403, gin.H{"error": "no permission"})
			return
		}
	}

	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var node models.Node
	if err := tx.First(&node, id).Error; err != nil {
		tx.Rollback()
		c.JSON(404, gin.H{"error": "node not found"})
		return
	}

	// 记录删除前的快照以便审计
	oldVal := map[string]interface{}{
		"id":             node.ID,
		"name":           node.Name,
		"ip_address":     node.IPAddress,
		"last_heartbeat": node.LastHeartbeat,
		"created_at":     node.CreatedAt,
		"updated_at":     node.UpdatedAt,
	}

	// 将该节点下的规则置为失效（is_active = false）
	if err := tx.Model(&models.RuleGroup{}).
		Where("node_id = ?", node.ID).
		Updates(map[string]interface{}{"is_active": false, "updated_at": time.Now()}).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "failed to deactivate related rules: " + err.Error()})
		return
	}

	// 删除节点记录
	if err := tx.Delete(&models.Node{}, node.ID).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "failed to delete node: " + err.Error()})
		return
	}

	// 记录审计日志（节点删除）
	h.createNodeAuditLog(c, uid, node.ID, node.Name, "delete", oldVal, nil,
		fmt.Sprintf("删除节点并将其下规则置为失效: %s (%s)", node.Name, node.IPAddress))

	tx.Commit()
	c.JSON(200, gin.H{"message": "node deleted"})
}

// ---- permission helpers (node) ----
func (h *AgentHandler) isAdmin(userID int) bool {
	var role models.UserRole
	err := h.DB.Where("user_id = ?", userID).First(&role).Error
	if err != nil {
		// 没有角色记录，默认为普通用户
		return false
	}
	return role.Role == "admin"
}

func (h *AgentHandler) hasNodePermission(userID int, nodeID int, action string) bool {
	if h.isAdmin(userID) {
		return true
	}
	var perm models.Permission
	if action == "read" {
		err := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action IN ?",
			userID, "node", nodeID, []string{"read", "write"}).First(&perm).Error
		return err == nil
	}
	err := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action = ?",
		userID, "node", nodeID, "write").First(&perm).Error
	return err == nil
}

// UpdateNodeTags updates the tags for a specific node.
func (h *AgentHandler) UpdateNodeTags(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)

	// Permission: need write on this node or admin
	if !h.hasNodePermission(uid, id, "write") {
		c.JSON(http.StatusForbidden, gin.H{"error": "no write permission"})
		return
	}

	var req struct {
		Tags []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var node models.Node
	if err := tx.Preload("Tags").First(&node, id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	// For audit log
	oldTagNames := make([]string, len(node.Tags))
	for i, tag := range node.Tags {
		oldTagNames[i] = tag.Name
	}

	processedTags, err := h.processTags(req.Tags)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process tags", "details": err.Error()})
		return
	}

	if err := tx.Model(&node).Association("Tags").Replace(processedTags); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update node tags", "details": err.Error()})
		return
	}

	// Audit log
	h.createNodeAuditLog(c, uid, node.ID, node.Name, "update_tags", oldTagNames, req.Tags,
		fmt.Sprintf("更新节点标签: %s", node.Name))

	tx.Commit()

	// Return the updated node with tags
	var updatedNode models.Node
	if err := h.DB.Preload("Tags").First(&updatedNode, id).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "Node tags updated successfully, but failed to retrieve updated node."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node tags updated successfully", "data": updatedNode})
}

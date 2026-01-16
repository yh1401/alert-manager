package handlers

import (
	"alert-manager-backend/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	BaseHandler
}

// CreateAuditLog 创建审计日志记录
func (h *AuditHandler) CreateAuditLog(c *gin.Context, userID int, username string, resourceType string, resourceID int, resourceName string, action string, oldValue interface{}, newValue interface{}, description string) error {
	// 将旧值和新值序列化为 JSON
	oldJSON, _ := json.Marshal(oldValue)
	newJSON, _ := json.Marshal(newValue)

	// 获取客户端 IP
	ipAddress := c.ClientIP()

	log := models.AuditLog{
		UserID:       userID,
		Username:     username,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Action:       action,
		OldValue:     string(oldJSON),
		NewValue:     string(newJSON),
		Description:  description,
		IPAddress:    ipAddress,
		CreatedAt:    time.Now(),
	}

	return h.DB.Create(&log).Error
}

// GetAuditLogs 获取审计日志列表（管理员专用）
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
	// 验证管理员权限
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	uid := uidVal.(int)

	var role models.UserRole
	if err := h.DB.Where("user_id = ?", uid).First(&role).Error; err != nil || role.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可查看审计日志"})
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resourceType := c.Query("resource_type") // 可选：rule / node
	action := c.Query("action")              // 可选：create / update / delete / rollback
	username := c.Query("username")          // 可选：按用户名筛选
	startDate := c.Query("start_date")       // 可选：开始日期
	endDate := c.Query("end_date")           // 可选：结束日期

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 构建查询
	query := h.DB.Model(&models.AuditLog{})

	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 获取分页数据
	var logs []models.AuditLog
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取审计日志失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetAuditLogDetail 获取单条审计日志详情（包含完整的旧值/新值对比）
func (h *AuditHandler) GetAuditLogDetail(c *gin.Context) {
	// 验证管理员权限
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	uid := uidVal.(int)

	var role models.UserRole
	if err := h.DB.Where("user_id = ?", uid).First(&role).Error; err != nil || role.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可查看审计日志"})
		return
	}

	// 获取日志ID
	logID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的日志ID"})
		return
	}

	var log models.AuditLog
	if err := h.DB.First(&log, logID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审计日志不存在"})
		return
	}

	// 解析 JSON 值以便前端展示
	var oldValue, newValue interface{}
	if log.OldValue != "" {
		json.Unmarshal([]byte(log.OldValue), &oldValue)
	}
	if log.NewValue != "" {
		json.Unmarshal([]byte(log.NewValue), &newValue)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"id":            log.ID,
			"user_id":       log.UserID,
			"username":      log.Username,
			"resource_type": log.ResourceType,
			"resource_id":   log.ResourceID,
			"resource_name": log.ResourceName,
			"action":        log.Action,
			"old_value":     oldValue,
			"new_value":     newValue,
			"description":   log.Description,
			"ip_address":    log.IPAddress,
			"created_at":    log.CreatedAt,
		},
	})
}

// GetAuditStats 获取审计统计数据（可选：用于仪表盘）
func (h *AuditHandler) GetAuditStats(c *gin.Context) {
	// 验证管理员权限
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	uid := uidVal.(int)

	var role models.UserRole
	if err := h.DB.Where("user_id = ?", uid).First(&role).Error; err != nil || role.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可查看审计统计"})
		return
	}

	// 统计各类操作数量
	type StatResult struct {
		Action string
		Count  int64
	}

	var actionStats []StatResult
	h.DB.Model(&models.AuditLog{}).
		Select("action, COUNT(*) as count").
		Group("action").
		Find(&actionStats)

	var resourceTypeStats []StatResult
	h.DB.Model(&models.AuditLog{}).
		Select("resource_type, COUNT(*) as count").
		Group("resource_type").
		Find(&resourceTypeStats)

	// 最近7天的操作趋势
	var dailyStats []struct {
		Date  string
		Count int64
	}
	h.DB.Model(&models.AuditLog{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= NOW() - INTERVAL '7 days'").
		Group("DATE(created_at)").
		Order("date").
		Find(&dailyStats)

	// 操作最频繁的用户（TOP 10）
	var topUsers []struct {
		Username string
		Count    int64
	}
	h.DB.Model(&models.AuditLog{}).
		Select("username, COUNT(*) as count").
		Group("username").
		Order("count DESC").
		Limit(10).
		Find(&topUsers)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"action_stats":        actionStats,
			"resource_type_stats": resourceTypeStats,
			"daily_stats":         dailyStats,
			"top_users":           topUsers,
		},
	})
}

// RestoreDeletedRule 从审计日志恢复被删除的规则（仅管理员）
func (h *AuditHandler) RestoreDeletedRule(c *gin.Context) {
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	uid := uidVal.(int)

	var role models.UserRole
	if err := h.DB.Where("user_id = ?", uid).First(&role).Error; err != nil || role.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可执行恢复"})
		return
	}

	var req struct {
		AuditID int `json:"audit_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var log models.AuditLog
	if err := h.DB.First(&log, req.AuditID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "审计记录不存在"})
		return
	}
	if log.ResourceType != "rule" || log.Action != "delete" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持从规则删除记录恢复"})
		return
	}

	var oldData struct {
		NodeID      int    `json:"node_id"`
		FilePath    string `json:"file_path"`
		Name        string `json:"name"`
		FileContent string `json:"file_content"`
		Version     int    `json:"version"`
		IsActive    bool   `json:"is_active"`
	}
	if err := json.Unmarshal([]byte(log.OldValue), &oldData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "旧值解析失败"})
		return
	}
	if oldData.Name == "" || oldData.FileContent == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "旧值缺少必要字段"})
		return
	}

	version := oldData.Version
	if version < 1 {
		version = 1
	}
	isActive := oldData.IsActive
	if !oldData.IsActive {
		isActive = true
	}

	rule := models.RuleGroup{
		NodeID:      oldData.NodeID,
		FilePath:    oldData.FilePath,
		Name:        oldData.Name,
		FileContent: oldData.FileContent,
		IsActive:    isActive,
		Version:     version,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx := h.DB.Begin()
	if err := tx.Create(&rule).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复规则失败: " + err.Error()})
		return
	}

	username := log.Username
	if username == "" {
		var user models.User
		if err := h.DB.First(&user, uid).Error; err == nil {
			username = user.Username
		} else {
			username = "unknown"
		}
	}

	if err := h.CreateAuditLog(c, uid, username, "rule", rule.ID, rule.Name, "rollback", nil, map[string]interface{}{
		"node_id":             rule.NodeID,
		"file_path":           rule.FilePath,
		"name":                rule.Name,
		"file_content":        rule.FileContent,
		"version":             rule.Version,
		"restored_from_audit": req.AuditID,
	}, fmt.Sprintf("从审计日志恢复规则: %s (audit_id=%d)", rule.Name, req.AuditID)); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "审计记录写入失败: " + err.Error()})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message": "规则已从审计记录恢复",
		"rule_id": rule.ID,
		"version": rule.Version,
	})
}

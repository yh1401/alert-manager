package handlers

import (
	"alert-manager-backend/models"
	"alert-manager-backend/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pmezard/go-difflib/difflib"
)

type RuleHandler struct {
	BaseHandler
}

// helper: get username by userID
func (h *RuleHandler) getUsername(userID int) string {
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return "unknown"
	}
	return user.Username
}

// helper: create audit log for rule operations
func (h *RuleHandler) createRuleAuditLog(c *gin.Context, userID int, resourceID int, resourceName string, action string, oldValue interface{}, newValue interface{}, description string) {
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

// helper: check if user is admin
func (h *RuleHandler) isAdmin(userID int) bool {
	var role models.UserRole
	err := h.DB.Where("user_id = ?", userID).First(&role).Error
	if err != nil {
		// 没有角色记录，默认为普通用户
		return false
	}
	return role.Role == "admin"
}

// helper: check if user has permission on a rule
// action: "read" or "write"; for read, write permission also suffices
func (h *RuleHandler) hasRulePermission(userID int, ruleID int, action string) bool {
	// admin bypass
	if h.isAdmin(userID) {
		return true
	}
	var perm models.Permission
	if action == "read" {
		// read is satisfied by read or write
		err := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action IN ?",
			userID, "rule", ruleID, []string{"read", "write"}).First(&perm).Error
		return err == nil
	}
	// write must be explicit write
	err := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action = ?",
		userID, "rule", ruleID, "write").First(&perm).Error
	return err == nil
}

func validateFilePath(p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("file_path is required")
	}
	if !filepath.IsAbs(p) {
		return fmt.Errorf("file_path must be absolute")
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("file_path must not contain '..'")
	}
	return nil
}

// CreateRule 创建新的告警规则
func (h *RuleHandler) CreateRule(c *gin.Context) {
	// only admin can create rules
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)
	if !h.isAdmin(uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can create rules"})
		return
	}
	var req struct {
		NodeID      int      `json:"node_id" binding:"required"`
		FilePath    string   `json:"file_path" binding:"required"`
		Name        string   `json:"name" binding:"required"`
		FileContent string   `json:"file_content" binding:"required"` // YAML 内容
		Tags        []string `json:"tags"`                            // 标签
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateFilePath(req.FilePath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 处理标签
	tags, err := h.processTags(req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process tags", "details": err.Error()})
		return
	}

	newRule := models.RuleGroup{
		NodeID:      req.NodeID,
		FilePath:    req.FilePath,
		Name:        req.Name,
		FileContent: req.FileContent,
		IsActive:    true, // 默认启用
		Version:     1,    // 初始版本号
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tags:        tags,
	}

	if err := h.DB.Create(&newRule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule: " + err.Error()})
		return
	}

	// 记录审计日志
	h.createRuleAuditLog(c, uid, newRule.ID, newRule.Name, "create", nil, map[string]interface{}{
		"node_id":      newRule.NodeID,
		"file_path":    newRule.FilePath,
		"name":         newRule.Name,
		"file_content": newRule.FileContent,
		"is_active":    newRule.IsActive,
		"version":      newRule.Version,
		"tags":         req.Tags,
	}, fmt.Sprintf("创建规则: %s (节点: %d, 路径: %s)", newRule.Name, newRule.NodeID, newRule.FilePath))

	c.JSON(http.StatusOK, gin.H{"message": "Rule created successfully", "id": newRule.ID})
}

// UpdateRule 更新告警规则 (带版本控制 + 三方合并)
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	var req struct {
		ID          int       `json:"id" binding:"required"`
		NodeID      int       `json:"node_id"`   // 可选：更新所属节点
		FilePath    string    `json:"file_path"` // 可选：更新目标文件路径
		Name        string    `json:"name"`
		FileContent string    `json:"file_content"`
		Comment     string    `json:"comment"`
		BaseVersion int       `json:"base_version" binding:"required"` // 乐观锁基线版本
		Tags        *[]string `json:"tags"`                            // 使用指针以区分 "未提供" 和 "清空"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果前端传入 file_path，先做服务端校验（必须为绝对路径且不包含 ..）
	if strings.TrimSpace(req.FilePath) != "" {
		if err := validateFilePath(req.FilePath); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// permission: need write on this rule or admin
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)
	if !h.hasRulePermission(uid, req.ID, "write") {
		c.JSON(http.StatusForbidden, gin.H{"error": "no write permission"})
		return
	}

	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var rule models.RuleGroup
	// Preload tags to get old tags for audit log
	if err := tx.Preload("Tags").First(&rule, req.ID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// For audit log
	oldTagNames := make([]string, len(rule.Tags))
	for i, tag := range rule.Tags {
		oldTagNames[i] = tag.Name
	}

	// 预先获取 base 版本内容，用于后续三方合并
	var baseContent string
	var baseFound bool
	if req.BaseVersion == int(rule.Version) {
		// base_version == 当前版本，base 就是当前内容
		baseContent = rule.FileContent
		baseFound = true
	} else if req.BaseVersion > 0 {
		// 从历史版本中取 base 内容
		var baseHistory models.RuleGroupVersion
		if err := tx.Where("rule_group_id = ? AND version = ?", rule.ID, req.BaseVersion).First(&baseHistory).Error; err == nil {
			baseContent = baseHistory.FileContent
			baseFound = true
		}
	}

	// 乐观锁校验：如果版本不一致，尝试三方合并；合并失败再报冲突
	if req.BaseVersion > 0 && req.BaseVersion != int(rule.Version) {
		latestContent := rule.FileContent
		clientContent := req.FileContent
		if clientContent == "" {
			clientContent = latestContent
		}

		if baseFound {
			merged, err := utils.ThreeWayMergeLines(baseContent, latestContent, clientContent)
			if err == nil {
				// 三方合并成功：把合并结果作为新内容继续走更新流程
				req.FileContent = merged
			} else {
				// 三方合并失败：返回详细冲突信息和 diff，提示用户手动解决
				ud := difflib.UnifiedDiff{
					A:        difflib.SplitLines(latestContent),
					B:        difflib.SplitLines(clientContent),
					FromFile: fmt.Sprintf("latest(v%d)", rule.Version),
					ToFile:   fmt.Sprintf("yours(base v%d)", req.BaseVersion),
					Context:  3,
				}

				var buf bytes.Buffer
				_ = difflib.WriteUnifiedDiff(&buf, ud)

				tx.Rollback()
				c.JSON(http.StatusConflict, gin.H{
					"error":             "version conflict, please refresh",
					"latest_version":    rule.Version,
					"your_base_version": req.BaseVersion,
					"latest_content":    latestContent,
					"diff":              buf.String(),
					"merge_error":       err.Error(),
				})
				return
			}
		} else {
			// 找不到 base 内容，无法做三方合并，退化为原来的冲突处理
			ud := difflib.UnifiedDiff{
				A:        difflib.SplitLines(latestContent),
				B:        difflib.SplitLines(clientContent),
				FromFile: fmt.Sprintf("latest(v%d)", rule.Version),
				ToFile:   fmt.Sprintf("yours(base v%d)", req.BaseVersion),
				Context:  3,
			}

			var buf bytes.Buffer
			_ = difflib.WriteUnifiedDiff(&buf, ud)

			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{
				"error":             "version conflict, please refresh",
				"latest_version":    rule.Version,
				"your_base_version": req.BaseVersion,
				"latest_content":    latestContent,
				"diff":              buf.String(),
				"merge_error":       "base version content not found, cannot auto-merge",
			})
			return
		}
	}

	// 1. 保存历史版本 (快照)
	history := models.RuleGroupVersion{
		RuleGroupID: rule.ID,
		NodeID:      rule.NodeID,
		FilePath:    rule.FilePath,
		Name:        rule.Name,
		FileContent: rule.FileContent,
		Version:     rule.Version,
		Comment:     req.Comment,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save history: " + err.Error()})
		return
	}

	// 2. 更新主表（构建新值）
	oldName := rule.Name
	oldContent := rule.FileContent
	oldVersion := rule.Version
	oldNodeID := rule.NodeID
	oldFilePath := rule.FilePath

	newName := rule.Name
	if req.Name != "" {
		newName = req.Name
	}
	newContent := rule.FileContent
	if req.FileContent != "" {
		newContent = req.FileContent
	}
	// 处理 node_id 与 file_path 的变更（如果前端未提供则保持原值）
	newNodeID := rule.NodeID
	if req.NodeID != 0 {
		newNodeID = req.NodeID
	}
	newFilePath := rule.FilePath
	if strings.TrimSpace(req.FilePath) != "" {
		newFilePath = req.FilePath
	}

	newVersion := rule.Version + 1

	updates := map[string]interface{}{
		"name":         newName,
		"file_content": newContent,
		"version":      newVersion,
		"updated_at":   time.Now(),
		"node_id":      newNodeID,
		"file_path":    newFilePath,
	}

	result := tx.Model(&rule).Where("version = ?", oldVersion).Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		// 版本在更新时已变更，触发冲突（此处不再尝试自动合并，避免无限重试）
		var latest models.RuleGroup
		_ = tx.First(&latest, rule.ID).Error
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{
			"error":             "version conflict, please refresh",
			"latest_version":    latest.Version,
			"your_base_version": req.BaseVersion,
			"latest_content":    latest.FileContent,
		})
		return
	}

	// 3. 更新标签 (如果提供了 tags 字段)
	var newTagNames []string
	if req.Tags != nil {
		processedTags, err := h.processTags(*req.Tags)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process tags", "details": err.Error()})
			return
		}

		if err := tx.Model(&rule).Association("Tags").Replace(processedTags); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule tags", "details": err.Error()})
			return
		}
		newTagNames = *req.Tags
	} else {
		// If tags field is not provided, keep old tags
		newTagNames = oldTagNames
	}

	// 为保证审计记录使用的是数据库最终持久化的值，重新从数据库读取已更新的记录
	var persisted models.RuleGroup
	if err := tx.First(&persisted, rule.ID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated rule: " + err.Error()})
		return
	}

	// 记录审计日志（使用持久化后的新值）
	h.createRuleAuditLog(c, uid, persisted.ID, persisted.Name, "update", map[string]interface{}{
		"name":         oldName,
		"file_content": oldContent,
		"version":      oldVersion,
		"node_id":      oldNodeID,
		"file_path":    oldFilePath,
		"tags":         oldTagNames,
	}, map[string]interface{}{
		"name":         persisted.Name,
		"file_content": persisted.FileContent,
		"version":      persisted.Version,
		"node_id":      persisted.NodeID,
		"file_path":    persisted.FilePath,
		"comment":      req.Comment,
		"tags":         newTagNames,
	}, fmt.Sprintf("更新规则: %s (版本: %d -> %d) (节点: %d -> %d, 路径: %s -> %s)", persisted.Name, oldVersion, persisted.Version, oldNodeID, persisted.NodeID, oldFilePath, persisted.FilePath))

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Rule updated successfully", "version": persisted.Version})
}

func (h *RuleHandler) ValidateRule(c *gin.Context) {
	var req struct {
		RuleID      int    `json:"rule_id"`                         // 可选：校验已有规则时带上，用于权限校验
		FileContent string `json:"file_content" binding:"required"` // 规则 YAML 内容
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)

	// 权限：
	// - 如果携带 rule_id，则需要对应规则的写权限
	// - 如果未携带 rule_id，则仅管理员可校验（用于新建规则场景）
	if req.RuleID > 0 {
		if !h.hasRulePermission(uid, req.RuleID, "write") {
			c.JSON(http.StatusForbidden, gin.H{"error": "no write permission"})
			return
		}
	} else if !h.isAdmin(uid) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can validate new rules"})
		return
	}

	promtoolPath := os.Getenv("PROMTOOL_PATH")
	if promtoolPath == "" {
		promtoolPath = "./tools/promtool/promtool"
	}

	output, err := utils.ValidateRulesWithPromtool(promtoolPath, req.FileContent)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"output":  output,
			"tool":    promtoolPath,
			"message": "promtool validation failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "规则语法校验通过",
		"output":  output,
		"tool":    promtoolPath,
	})
}

// GetRuleList 获取规则列表
func (h *RuleHandler) GetRuleList(c *gin.Context) {
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)

	var rules []models.RuleGroup
	var err error

	if h.isAdmin(uid) {
		err = h.DB.Preload("Tags").Order("updated_at desc").Find(&rules).Error
	} else {
		// not admin: filter by permissions
		var permIDs []int
		type Row struct{ ResourceID int }
		var rows []Row
		if err := h.DB.Model(&models.Permission{}).
			Select("resource_id").
			Where("user_id = ? AND resource_type = ? AND action IN ?", uid, "rule", []string{"read", "write"}).
			Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch permissions"})
			return
		}
		for _, r := range rows {
			permIDs = append(permIDs, r.ResourceID)
		}
		if len(permIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []models.RuleGroup{}})
			return
		}
		err = h.DB.Preload("Tags").Where("id IN ?", permIDs).Order("updated_at desc").Find(&rules).Error
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rules"})
		return
	}

	// 为前端转换数据结构
	type RuleGroupResponse struct {
		ID          int       `json:"id"`
		NodeID      int       `json:"node_id"`
		FilePath    string    `json:"file_path"`
		Name        string    `json:"name"`
		FileContent string    `json:"file_content"`
		IsActive    bool      `json:"is_active"`
		Version     int       `json:"version"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Tags        []string  `json:"tags"`
	}

	response := make([]RuleGroupResponse, len(rules))
	for i, rule := range rules {
		tagNames := make([]string, len(rule.Tags))
		for j, tag := range rule.Tags {
			tagNames[j] = tag.Name
		}

		response[i] = RuleGroupResponse{
			ID:          rule.ID,
			NodeID:      rule.NodeID,
			FilePath:    rule.FilePath,
			Name:        rule.Name,
			FileContent: rule.FileContent,
			IsActive:    rule.IsActive,
			Version:     rule.Version,
			CreatedAt:   rule.CreatedAt,
			UpdatedAt:   rule.UpdatedAt,
			Tags:        tagNames,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// DeleteRule 删除规则
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	var req struct {
		ID int `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// permission: need write on this rule or admin
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int)
	if !h.hasRulePermission(uid, req.ID, "write") {
		c.JSON(http.StatusForbidden, gin.H{"error": "no write permission"})
		return
	}

	// 先获取规则信息用于审计日志
	var rule models.RuleGroup
	if err := h.DB.First(&rule, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	if err := h.DB.Delete(&models.RuleGroup{}, req.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}

	// 记录审计日志
	h.createRuleAuditLog(c, uid, rule.ID, rule.Name, "delete", map[string]interface{}{
		"node_id":      rule.NodeID,
		"file_path":    rule.FilePath,
		"name":         rule.Name,
		"file_content": rule.FileContent,
		"version":      rule.Version,
	}, nil, fmt.Sprintf("删除规则: %s (节点: %d, 路径: %s)", rule.Name, rule.NodeID, rule.FilePath))

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

// GetRuleVersions 获取规则的版本历史
func (h *RuleHandler) GetRuleVersions(c *gin.Context) {
	var req struct {
		ID int `form:"id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// read permission required
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if !h.hasRulePermission(uidVal.(int), req.ID, "read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "no read permission"})
		return
	}

	// 检查规则是否存在
	var rule models.RuleGroup
	if err := h.DB.First(&rule, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// 获取所有版本历史，按创建时间降序
	var versions []models.RuleGroupVersion
	if err := h.DB.Where("rule_group_id = ?", req.ID).Order("version desc").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch versions"})
		return
	}

	// 包含当前版本信息
	currentVersion := map[string]interface{}{
		"id":            rule.ID,
		"rule_group_id": rule.ID,
		"version":       rule.Version,
		"name":          rule.Name,
		"node_id":       rule.NodeID,
		"file_path":     rule.FilePath,
		"file_content":  rule.FileContent,
		"is_current":    true,
		"created_at":    rule.UpdatedAt,
		"comment":       "[Current Version]",
	}

	// 构建响应数据
	allVersions := []interface{}{currentVersion}
	for _, v := range versions {
		allVersions = append(allVersions, gin.H{
			"id":            v.ID,
			"rule_group_id": v.RuleGroupID,
			"version":       v.Version,
			"name":          v.Name,
			"node_id":       v.NodeID,
			"file_path":     v.FilePath,
			"file_content":  v.FileContent,
			"is_current":    false,
			"created_at":    v.CreatedAt,
			"comment":       v.Comment,
			"created_by":    v.CreatedBy,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": allVersions})
}

// RollbackRule 回滚到指定版本：追加一个新版本，内容与目标版本一致
func (h *RuleHandler) RollbackRule(c *gin.Context) {
	var req struct {
		ID      int    `json:"id" binding:"required"`      // rule_group_id
		Version int    `json:"version" binding:"required"` // 要回滚的版本号
		Comment string `json:"comment"`                    // 回滚说明
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[Rollback] Bind error: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// permission: need write
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if !h.hasRulePermission(uidVal.(int), req.ID, "write") {
		c.JSON(http.StatusForbidden, gin.H{"error": "no write permission"})
		return
	}

	if req.Comment == "" {
		req.Comment = "Rollback to version " + fmt.Sprintf("%d", req.Version)
	}

	tx := h.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取当前规则
	var rule models.RuleGroup
	if err := tx.First(&rule, req.ID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// 获取目标版本内容：允许回滚到历史版本，或当前版本号（相当于不变）
	var targetContent models.RuleGroupVersion
	var content string
	var name string

	if req.Version == rule.Version {
		// 目标是当前版本
		content = rule.FileContent
		name = rule.Name
	} else {
		if err := tx.Where("rule_group_id = ? AND version = ?", req.ID, req.Version).First(&targetContent).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Target version not found"})
			return
		}
		content = targetContent.FileContent
		name = targetContent.Name
	}

	// 1) 先把当前版本存历史，作为回滚前快照
	backup := models.RuleGroupVersion{
		RuleGroupID: rule.ID,
		NodeID:      rule.NodeID,
		FilePath:    rule.FilePath,
		Name:        rule.Name,
		FileContent: rule.FileContent,
		Version:     rule.Version,
		Comment:     req.Comment,
		CreatedAt:   time.Now(),
	}
	if err := tx.Create(&backup).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save rollback history: " + err.Error()})
		return
	}

	// 2) 计算新版本号（注意：不需要将新版本立即写入历史表，只有在下次更新时才会归档当前版本）
	newVersionNumber := rule.Version + 1

	// 3) 更新主表到新版本
	updates := map[string]interface{}{
		"version":      newVersionNumber,
		"file_content": content,
		"name":         name,
		"updated_at":   time.Now(),
	}
	if err := tx.Model(&rule).Updates(updates).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rollback rule: " + err.Error()})
		return
	}

	// 记录审计日志
	uid := uidVal.(int)
	h.createRuleAuditLog(c, uid, rule.ID, rule.Name, "rollback", map[string]interface{}{
		"version":      rule.Version,
		"file_content": rule.FileContent,
	}, map[string]interface{}{
		"version":         newVersionNumber,
		"file_content":    content,
		"rollback_target": req.Version,
	}, fmt.Sprintf("回滚规则: %s (版本: %d -> %d, 回滚至版本 %d)", rule.Name, rule.Version, newVersionNumber, req.Version))

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"message": "Rollback created new version",
		"version": newVersionNumber,
	})
}

// GetRuleVersionDiff 获取两个版本的差异
func (h *RuleHandler) GetRuleVersionDiff(c *gin.Context) {
	var req struct {
		ID          int `form:"id" binding:"required"`           // rule_group_id
		FromVersion int `form:"from_version" binding:"required"` // 源版本
		ToVersion   int `form:"to_version" binding:"required"`   // 目标版本
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// read permission required
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if !h.hasRulePermission(uidVal.(int), req.ID, "read") {
		c.JSON(http.StatusForbidden, gin.H{"error": "no read permission"})
		return
	}

	// 检查规则是否存在
	var rule models.RuleGroup
	if err := h.DB.First(&rule, req.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// 获取源版本内容
	var fromContent string
	if req.FromVersion == rule.Version {
		// 如果是当前版本
		fromContent = rule.FileContent
	} else {
		var fromVersion models.RuleGroupVersion
		if err := h.DB.Where("rule_group_id = ? AND version = ?", req.ID, req.FromVersion).First(&fromVersion).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "From version not found"})
			return
		}
		fromContent = fromVersion.FileContent
	}

	// 获取目标版本内容
	var toContent string
	if req.ToVersion == rule.Version {
		// 如果是当前版本
		toContent = rule.FileContent
	} else {
		var toVersion models.RuleGroupVersion
		if err := h.DB.Where("rule_group_id = ? AND version = ?", req.ID, req.ToVersion).First(&toVersion).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "To version not found"})
			return
		}
		toContent = toVersion.FileContent
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"from_version": req.FromVersion,
			"to_version":   req.ToVersion,
			"from_content": fromContent,
			"to_content":   toContent,
		},
	})
}

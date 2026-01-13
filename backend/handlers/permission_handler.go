package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"alert-manager-backend/models"
)

type PermissionHandler struct {
	BaseHandler
}

// HasPermission 检查用户是否对资源有指定权限
func (h *PermissionHandler) HasPermission(userID int, resourceType string, resourceID int, action string) bool {
	var perm models.Permission
	err := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action = ?",
		userID, resourceType, resourceID, action).First(&perm).Error
	return err == nil
}

// helper function to check if user is admin
func (h *PermissionHandler) isAdmin(userID int) bool {
	var role models.UserRole
	err := h.DB.Where("user_id = ?", userID).First(&role).Error
	if err != nil {
		// 没有角色记录，默认为普通用户
		return false
	}
	return role.Role == "admin"
}

// SetPermission 设置用户权限（仅管理员）
func (h *PermissionHandler) SetPermission(c *gin.Context) {
	var req struct {
		UserID       int    `json:"user_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"` // rule / node
		ResourceID   int    `json:"resource_id" binding:"required"`
		Action       string `json:"action" binding:"required"` // read / write
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, ok := c.Get("userID")
	if !ok || !h.isAdmin(adminID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can set permissions"})
		return
	}
	perm := models.Permission{
		UserID:       req.UserID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		Action:       req.Action,
	}
	h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action = ?",
		req.UserID, req.ResourceType, req.ResourceID, req.Action).FirstOrCreate(&perm)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RemovePermission 移除用户权限（仅管理员）
func (h *PermissionHandler) RemovePermission(c *gin.Context) {
	var req struct {
		UserID       int    `json:"user_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceID   int    `json:"resource_id" binding:"required"`
		Action       string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, ok := c.Get("userID")
	if !ok || !h.isAdmin(adminID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can remove permissions"})
		return
	}
	h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action = ?",
		req.UserID, req.ResourceType, req.ResourceID, req.Action).Delete(&models.Permission{})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetUserPermissions 获取当前用户的所有权限
func (h *PermissionHandler) GetUserPermissions(c *gin.Context) {
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	var perms []models.Permission
	h.DB.Where("user_id = ?", userID).Find(&perms)
	c.JSON(http.StatusOK, gin.H{"data": perms})
}

// SetUserRole 设置用户角色（仅管理员）
func (h *PermissionHandler) SetUserRole(c *gin.Context) {
	var req struct {
		UserID int    `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required"` // admin / user
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, ok := c.Get("userID")
	if !ok || !h.isAdmin(adminID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can set roles"})
		return
	}
	var role models.UserRole
	if err := h.DB.Where("user_id = ?", req.UserID).First(&role).Error; err != nil {
		role = models.UserRole{UserID: req.UserID, Role: req.Role}
		h.DB.Create(&role)
	} else {
		h.DB.Model(&role).Update("role", req.Role)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ListUsers 列出所有用户及其角色（仅管理员）
func (h *PermissionHandler) ListUsers(c *gin.Context) {
	adminID, ok := c.Get("userID")
	if !ok || !h.isAdmin(adminID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can list users"})
		return
	}
	var users []models.User
	h.DB.Find(&users)
	type UserInfo struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	res := make([]UserInfo, 0, len(users))
	for _, u := range users {
		var role models.UserRole
		roleStr := "user"
		if err := h.DB.Where("user_id = ?", u.ID).First(&role).Error; err == nil {
			roleStr = role.Role
		}
		res = append(res, UserInfo{ID: u.ID, Username: u.Username, Role: roleStr})
	}
	c.JSON(http.StatusOK, gin.H{"data": res})
}

// GetUserPermissionsByID 管理员或用户本人查看权限
func (h *PermissionHandler) GetUserPermissionsByID(c *gin.Context) {
	idStr := c.Param("user_id")
	uid, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	reqUserID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	if reqUserID.(int) != uid && !h.isAdmin(reqUserID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "no permission"})
		return
	}
	var perms []models.Permission
	h.DB.Where("user_id = ?", uid).Find(&perms)
	c.JSON(http.StatusOK, gin.H{"data": perms})
}

// BatchSetPermissions 批量设置权限（仅管理员）
func (h *PermissionHandler) BatchSetPermissions(c *gin.Context) {
	var req struct {
		UserID       int    `json:"user_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"` // rule / node
		ResourceIDs  []int  `json:"resource_ids" binding:"required"`  // 资源ID列表
		Action       string `json:"action" binding:"required"`        // read / write
		IncludeExist bool   `json:"include_exist"`                    // 可选：是否包括已存在的规则/节点
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, ok := c.Get("userID")
	if !ok || !h.isAdmin(adminID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can set permissions"})
		return
	}

	resourceIDs := req.ResourceIDs

	count := 0
	for _, rid := range resourceIDs {
		perm := models.Permission{
			UserID:       req.UserID,
			ResourceType: req.ResourceType,
			ResourceID:   rid,
			Action:       req.Action,
		}
		result := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id = ? AND action = ?",
			req.UserID, req.ResourceType, rid, req.Action).FirstOrCreate(&perm)
		if result.RowsAffected > 0 {
			count++
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "count": count, "total": len(resourceIDs)})
}

// BatchRemovePermissions 批量移除权限（仅管理员）
func (h *PermissionHandler) BatchRemovePermissions(c *gin.Context) {
	var req struct {
		UserID       int    `json:"user_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"`
		ResourceIDs  []int  `json:"resource_ids" binding:"required"`
		Action       string `json:"action" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, ok := c.Get("userID")
	if !ok || !h.isAdmin(adminID.(int)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only admin can remove permissions"})
		return
	}

	resourceIDs := req.ResourceIDs

	result := h.DB.Where("user_id = ? AND resource_type = ? AND resource_id IN ? AND action = ?",
		req.UserID, req.ResourceType, resourceIDs, req.Action).Delete(&models.Permission{})

	c.JSON(http.StatusOK, gin.H{"status": "ok", "count": result.RowsAffected})
}

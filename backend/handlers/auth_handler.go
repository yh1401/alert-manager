package handlers

import (
	"alert-manager-backend/models"
	"alert-manager-backend/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	BaseHandler
}

// Register 注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user := models.User{Username: req.Username, Password: string(hashedPwd)}

	result := h.DB.Create(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrDuplicatedKey {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已存在"})
			return
		}
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	// 为新用户自动创建默认角色（普通用户）
	userRole := models.UserRole{
		UserID: int(user.ID),
		Role:   "user",
	}
	if err := h.DB.Create(&userRole).Error; err != nil {
		fmt.Printf("创建用户角色失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "注册成功"})
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token, _ := utils.GenerateToken(user.ID, user.Username)
	// 返回格式适配大多数 Admin 模板
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{"access_token": token},
		"msg":  "登录成功",
	})
}

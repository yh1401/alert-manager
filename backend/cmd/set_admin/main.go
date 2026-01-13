package main

import (
	"alert-manager-backend/global"
	"alert-manager-backend/initialize"
	"alert-manager-backend/models"
	"fmt"
	"os"
	"strconv"
)

func main() {
	initialize.InitDB()

	if len(os.Args) < 3 {
		fmt.Println("用法: go run cmd/set_admin/main.go <user_id> <role>")
		fmt.Println("示例: go run cmd/set_admin/main.go 1 admin")
		fmt.Println("角色可选: admin / user")
		return
	}

	userID, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("错误: user_id 必须是数字")
		return
	}

	role := os.Args[2]
	if role != "admin" && role != "user" {
		fmt.Println("错误: 角色必须是 admin 或 user")
		return
	}

	// 检查用户是否存在
	var user models.User
	if err := global.DB.First(&user, userID).Error; err != nil {
		fmt.Printf("错误: 找不到 ID=%d 的用户\n", userID)
		return
	}

	// 检查是否已有角色记录
	var userRole models.UserRole
	err = global.DB.Where("user_id = ?", userID).First(&userRole).Error

	if err != nil {
		// 没有角色记录，创建新的
		userRole = models.UserRole{
			UserID: userID,
			Role:   role,
		}
		if err := global.DB.Create(&userRole).Error; err != nil {
			fmt.Printf("错误: 创建角色失败: %v\n", err)
			return
		}
		fmt.Printf("✓ 已为用户 %s (ID=%d) 设置角色为: %s\n", user.Username, userID, role)
	} else {
		// 已有角色记录，更新
		if err := global.DB.Model(&userRole).Update("role", role).Error; err != nil {
			fmt.Printf("错误: 更新角色失败: %v\n", err)
			return
		}
		fmt.Printf("✓ 已将用户 %s (ID=%d) 的角色更新为: %s\n", user.Username, userID, role)
	}
}

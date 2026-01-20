package main

import (
	"alert-manager-backend/global"
	"alert-manager-backend/initialize"
	"alert-manager-backend/models"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// main is a command-line tool to idempotently create or update an admin user.
func main() {
	initialize.InitDB()

	if len(os.Args) < 3 {
		fmt.Println("用法: go run cmd/setup_admin/main.go <username> <password>")
		fmt.Println("示例: go run cmd/setup_admin/main.go admin mysecretpassword")
		return
	}

	username := os.Args[1]
	password := os.Args[2]
	role := "admin"

	var user models.User
	err := global.DB.Where("username = ?", username).First(&user).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		fmt.Printf("错误: 查询用户失败: %v\n", err)
		return
	}

	// Case 1: User does not exist. Create user and role.
	if err == gorm.ErrRecordNotFound {
		fmt.Printf("用户 '%s' 不存在，正在创建...\n", username)

		hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println("错误: 密码加密失败")
			return
		}

		newUser := models.User{Username: username, Password: string(hashedPwd)}
		if err := global.DB.Create(&newUser).Error; err != nil {
			fmt.Printf("错误: 创建用户失败: %v\n", err)
			return
		}

		userRole := models.UserRole{
			UserID: int(newUser.ID),
			Role:   role,
		}
		if err := global.DB.Create(&userRole).Error; err != nil {
			fmt.Printf("错误: 为新用户创建角色失败: %v\n", err)
			// Attempt to roll back user creation for consistency
			global.DB.Delete(&newUser)
			return
		}
		fmt.Printf("✓ 用户 '%s' (ID=%d) 创建并设置为 '%s' 成功。\n", newUser.Username, newUser.ID, role)
		return
	}

	// Case 2: User exists. Check and update role if necessary.
	var userRole models.UserRole
	if err := global.DB.Where("user_id = ?", user.ID).First(&userRole).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// This case is unlikely if user exists, but we handle it.
			fmt.Printf("用户 '%s' 存在但没有角色，正在创建角色...\n", username)
			newUserRole := models.UserRole{UserID: int(user.ID), Role: role}
			if err := global.DB.Create(&newUserRole).Error; err != nil {
				fmt.Printf("错误: 为存在用户创建角色失败: %v\n", err)
				return
			}
			fmt.Printf("✓ 已为用户 '%s' (ID=%d) 设置角色为 '%s'。\n", user.Username, user.ID, role)
		} else {
			fmt.Printf("错误: 查询用户角色失败: %v\n", err)
		}
		return
	}

	if userRole.Role == role {
		fmt.Printf("✓ 用户 '%s' 已经是 '%s'，无需操作。\n", user.Username, role)
	} else {
		fmt.Printf("用户 '%s' 的角色为 '%s'，正在更新为 '%s'...\n", user.Username, userRole.Role, role)
		if err := global.DB.Model(&userRole).Update("role", role).Error; err != nil {
			fmt.Printf("错误: 更新角色失败: %v\n", err)
			return
		}
		fmt.Printf("✓ 已将用户 '%s' 的角色更新为 '%s'。\n", user.Username, role)
	}
}

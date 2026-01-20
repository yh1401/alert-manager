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

// main a command line tool to create a user
func main() {
	initialize.InitDB()

	if len(os.Args) < 3 {
		fmt.Println("用法: go run cmd/create_user/main.go <username> <password>")
		fmt.Println("示例: go run cmd/create_user/main.go admin mypassword")
		return
	}

	username := os.Args[1]
	password := os.Args[2]

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("错误: 密码加密失败")
		return
	}
	user := models.User{Username: username, Password: string(hashedPwd)}

	result := global.DB.Create(&user)

	if result.Error != nil {
		if result.Error == gorm.ErrDuplicatedKey {
			fmt.Println("错误: 用户名已存在")
			return
		}
		fmt.Printf("错误: 创建用户失败: %v\n", result.Error)
		return
	}

	// 为新用户自动创建默认角色（普通用户）
	userRole := models.UserRole{
		UserID: int(user.ID),
		Role:   "user",
	}
	if err := global.DB.Create(&userRole).Error; err != nil {
		fmt.Printf("创建用户角色失败: %v\n", err)
		// rollback user creation
		global.DB.Delete(&user)
		fmt.Println("已回滚用户创建。")
		return
	}

	fmt.Printf("✓ 用户 '%s' 创建成功 (ID=%d)\n", username, user.ID)
	fmt.Printf("! 请使用 'set_admin' 脚本为该用户分配管理员角色。\n")
	fmt.Printf("  示例: go run cmd/set_admin/main.go %d admin\n", user.ID)
}

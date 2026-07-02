package main

import (
	"log"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"alert-manager-backend/global"
	"alert-manager-backend/initialize"
	"alert-manager-backend/routes"
)

func main() {
	r := gin.Default()

	// 允许跨域请求 (CORS)，以便 Vue 前端可以调用接口
	r.Use(initialize.CorsMiddleware())

	// 1. 初始化 GORM (业务数据库连接)
	initialize.InitDB()

	// 2. 注册业务 API 路由
	routes.Register(r, global.DB)

	// 3. 启动服务
	log.Println("服务启动在 :30333")
	if err := r.Run(":30333"); err != nil {
		log.Fatal(err)
	}
}

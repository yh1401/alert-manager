package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"alert-manager-backend/handlers"
	"alert-manager-backend/middleware"
)

// Register 注册业务路由
func Register(r *gin.Engine, db *gorm.DB) {
	baseHandler := handlers.BaseHandler{DB: db}
	authHandler := &handlers.AuthHandler{BaseHandler: baseHandler}
	agentHandler := &handlers.AgentHandler{BaseHandler: baseHandler}
	ruleHandler := &handlers.RuleHandler{BaseHandler: baseHandler}
	permHandler := &handlers.PermissionHandler{BaseHandler: baseHandler}
	auditHandler := &handlers.AuditHandler{BaseHandler: baseHandler}
	tagHandler := &handlers.TagHandler{BaseHandler: baseHandler}

	api := r.Group("/api")
	{
		// Tag management endpoints
		tags := api.Group("/tags")
		tags.Use(middleware.AuthMiddleware())
		{
			tags.GET("", tagHandler.ListTags)
		}

		user := api.Group("/user")
		{
			user.POST("/login", authHandler.Login)
			user.POST("/register", authHandler.Register)
			userAuth := user.Group("")
			userAuth.Use(middleware.AuthMiddleware())
			{
				userAuth.GET("/permissions", permHandler.GetUserPermissions)
			}
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		{
			admin.GET("/users", permHandler.ListUsers)
			admin.POST("/permissions/set", permHandler.SetPermission)
			admin.POST("/permissions/remove", permHandler.RemovePermission)
			admin.POST("/permissions/batch-set", permHandler.BatchSetPermissions)
			admin.POST("/permissions/batch-remove", permHandler.BatchRemovePermissions)
			admin.POST("/users/role", permHandler.SetUserRole)
			admin.GET("/users/:user_id/permissions", permHandler.GetUserPermissionsByID)
			// 审计日志相关接口（仅管理员）
			admin.GET("/audit/logs", auditHandler.GetAuditLogs)                 // 获取审计日志列表
			admin.GET("/audit/logs/:id", auditHandler.GetAuditLogDetail)        // 获取单条审计日志详情
			admin.GET("/audit/stats", auditHandler.GetAuditStats)               // 获取审计统计数据
			admin.POST("/audit/rules/restore", auditHandler.RestoreDeletedRule) // 从审计删除记录恢复规则
		}

		rule := api.Group("/rule")
		rule.Use(middleware.AuthMiddleware())
		{
			rule.GET("/list", ruleHandler.GetRuleList)
			rule.POST("/validate_rule", ruleHandler.ValidateRule)
			rule.POST("/create_rule", ruleHandler.CreateRule)
			rule.POST("/update_rule", ruleHandler.UpdateRule)
			rule.POST("/delete_rule", ruleHandler.DeleteRule)
			// 版本管理相关接口
			rule.GET("/versions", ruleHandler.GetRuleVersions) // 获取版本历史
			rule.POST("/rollback", ruleHandler.RollbackRule)   // 回滚到指定版本
			rule.GET("/diff", ruleHandler.GetRuleVersionDiff)  // 获取版本差异
		}

		// public agent endpoints (called by agents, no JWT)
		agent := api.Group("/agent")
		{
			// 下载 agent 二进制，由后端提供静态/受控下载接口
			agent.GET("/download", agentHandler.ServeAgentBinary)

			agent.GET("/config_export", agentHandler.ExportConfig)
			agent.POST("/heartbeat", agentHandler.UpdateHeartbeat)
			agent.POST("/register", agentHandler.RegisterNode)

			// Agent 上报同步/重载状态（public, agent 调用）
			agent.POST("/report_sync", agentHandler.ReportSyncStatus)

			// UI endpoints require JWT
			nodes := agent.Group("")
			nodes.Use(middleware.AuthMiddleware())
			{
				nodes.GET("/nodes", agentHandler.ListNodes)
				nodes.GET("/nodes/:id", agentHandler.GetNodeDetail)
				nodes.GET("/nodes/:id/history", agentHandler.ListNodeSyncHistory)
				nodes.POST("/nodes/:id/manual_sync", agentHandler.ManualSync)
				nodes.POST("/nodes/:id/tags", agentHandler.UpdateNodeTags)
				nodes.DELETE("/nodes/:id", agentHandler.DeleteNode)
			}
		}
	}
}

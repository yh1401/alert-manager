package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"alert-manager-backend/global"
	"alert-manager-backend/prometheus"
)

// PrometheusHandler 处理 Prometheus 数据获取相关请求
type PrometheusHandler struct {
	BaseHandler
}

// getClient 获取全局 Prometheus 客户端
func (h *PrometheusHandler) getClient() (*prometheus.Client, error) {
	if global.PrometheusClient == nil {
		return nil, fmt.Errorf("Prometheus 客户端未初始化")
	}
	return global.PrometheusClient, nil
}

// Health 检查 Prometheus 服务健康状态
// GET /api/prometheus/health
func (h *PrometheusHandler) Health(c *gin.Context) {
	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "uninitialized", "error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unreachable",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": health.Status})
}

// GetAlerts 获取 Prometheus 活跃告警
// GET /api/prometheus/alerts
func (h *PrometheusHandler) GetAlerts(c *gin.Context) {
	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	alerts, err := client.AggregateAlerts(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      alerts,
		"total":     len(alerts),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetRules 获取 Prometheus 规则评估状态
// GET /api/prometheus/rules
func (h *PrometheusHandler) GetRules(c *gin.Context) {
	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	rules, err := client.AggregateRules(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      rules,
		"total":     len(rules),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetTargets 获取 Prometheus 采集目标健康状态
// GET /api/prometheus/targets
func (h *PrometheusHandler) GetTargets(c *gin.Context) {
	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	targets, err := client.AggregateTargets(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      targets,
		"total":     len(targets),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetOverview 获取 Prometheus 数据概览（用于仪表盘）
// GET /api/prometheus/overview
func (h *PrometheusHandler) GetOverview(c *gin.Context) {
	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	overview, err := client.GetOverview(ctx)
	if err != nil {
		c.JSON(http.StatusPartialContent, gin.H{
			"data":  overview,
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": overview})
}

// Query 执行即时 PromQL 查询
// GET /api/prometheus/query?query=up
func (h *PrometheusHandler) Query(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 query 参数"})
		return
	}

	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := client.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      result,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// QueryRange 执行范围 PromQL 查询
// GET /api/prometheus/query_range?query=up&start=...&end=...&step=15s
func (h *PrometheusHandler) QueryRange(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 query 参数"})
		return
	}

	startStr := c.Query("start")
	endStr := c.Query("end")
	step := c.DefaultQuery("step", "15s")

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		start, err = parseTimestamp(startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start 参数格式无效"})
			return
		}
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		end, err = parseTimestamp(endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "end 参数格式无效"})
			return
		}
	}

	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	result, err := client.QueryRange(ctx, query, start, end, step)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      result,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ClearCache 清除 Prometheus 数据缓存
// POST /api/prometheus/clear_cache
func (h *PrometheusHandler) ClearCache(c *gin.Context) {
	client, err := h.getClient()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	client.ClearCache()
	c.JSON(http.StatusOK, gin.H{"message": "缓存已清除"})
}

// parseTimestamp 尝试将字符串解析为 Unix 时间戳
func parseTimestamp(s string) (time.Time, error) {
	var ts int64
	_, err := fmt.Sscanf(s, "%d", &ts)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(ts, 0), nil
}

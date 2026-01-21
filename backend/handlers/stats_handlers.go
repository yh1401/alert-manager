package handlers

import (
	"alert-manager-backend/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	BaseHandler
}

type SyncFailureStat struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// GetNodeSyncFailureStats retrieves daily node sync failure statistics
func (h *StatsHandler) GetNodeSyncFailureStats(c *gin.Context) {
	// 验证管理员权限
	uidVal, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}
	uid := uidVal.(int)

	var role models.UserRole
	if err := h.DB.Where("user_id = ?", uid).First(&role).Error; err != nil || role.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅管理员可查看统计数据"})
		return
	}

	var stats []SyncFailureStat
	// Query for the last 30 days. The DB is postgres
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	result := h.DB.
		Model(&models.NodeSyncHistory{}).
		Select("DATE(created_at) as date, count(*) as count").
		Where("created_at >= ? AND (fetch_status = 'failed' OR reload_status = 'failed')", thirtyDaysAgo).
		Group("DATE(created_at)").
		Order("date ASC").
		Find(&stats)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sync failure stats"})
		return
	}

	// To ensure we have data for all last 30 days, we can fill in the gaps.
	// Create a map of date to count
	statsMap := make(map[string]int64)
	for _, s := range stats {
		// a bit of hack to handle the date format from postgres
		t, _ := time.Parse(time.RFC3339, s.Date)
		statsMap[t.Format("2006-01-02")] = s.Count
	}

	// Create a complete list for the last 30 days
	var completeStats []SyncFailureStat
	for i := 29; i >= 0; i-- {
		day := time.Now().AddDate(0, 0, -i)
		dateStr := day.Format("2006-01-02")
		count, ok := statsMap[dateStr]
		if !ok {
			count = 0
		}
		completeStats = append(completeStats, SyncFailureStat{Date: dateStr, Count: count})
	}

	c.JSON(http.StatusOK, gin.H{"data": completeStats})
}

package handlers

import (
	"alert-manager-backend/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// TagHandler handles tag-related API requests.
type TagHandler struct {
	BaseHandler
}

// ListTags retrieves all existing tags.
func (h *TagHandler) ListTags(c *gin.Context) {
	var tags []models.Tag
	if err := h.DB.Order("name asc").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tags", "details": err.Error()})
		return
	}

	if tags == nil {
		tags = make([]models.Tag, 0)
	}

	c.JSON(http.StatusOK, gin.H{"data": tags})
}

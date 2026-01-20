package handlers

import (
	"alert-manager-backend/models"
	"strings"

	"gorm.io/gorm"
)

type BaseHandler struct {
	DB *gorm.DB
}

// processTags finds or creates tags based on a list of names.
func (h *BaseHandler) processTags(tagNames []string) ([]*models.Tag, error) {
	if len(tagNames) == 0 {
		return nil, nil
	}

	var tags []*models.Tag
	// de-duplicate and trim
	uniqueTagNames := make(map[string]struct{})
	for _, tagName := range tagNames {
		trimmedTagName := strings.TrimSpace(tagName)
		if trimmedTagName != "" {
			uniqueTagNames[trimmedTagName] = struct{}{}
		}
	}

	for name := range uniqueTagNames {
		var tag models.Tag
		// Find existing tag or create a new one
		if err := h.DB.FirstOrCreate(&tag, models.Tag{Name: name}).Error; err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	return tags, nil
}

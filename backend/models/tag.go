// file: backend/models/tag.go
package models

import "time"

// Tag represents a tag for categorizing rules and nodes.
type Tag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:50" json:"name"` // Tag name, must be unique
	CreatedAt time.Time `json:"created_at"`
}

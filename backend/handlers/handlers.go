package handlers

import "gorm.io/gorm"

type BaseHandler struct {
	DB *gorm.DB
}

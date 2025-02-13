package model

import "gorm.io/gorm"

// Task task model
// @Description task table
type Task struct {
	gorm.Model // embeded gorm model
	JobID uint `gorm:"index"`
	Title string
	Creator string
	Status Status `gorm:"default:0"`
	ErrorCode ErrorCode `gorm:"default:0"`
	Config Config `gorm:"type:json"`
	Label []string `gorm:"type:json;default:'[]'"`
}
package model

import "gorm.io/gorm"


// Task task model
// @Description task table
type Task struct {
	gorm.Model // embeded gorm model
	JobID uint `gorm:"index"`
	Title string
	Creator string
	Status Status
	ErrorCode ErrorCode
	Config Config `gorm:"type:json"`
	Label []string `gorm:"type:json;default:'[]'::jsonb"`  // 修改为字符串切片并使用JSON类型存储
}
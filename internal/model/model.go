package model

import "gorm.io/gorm"

type Status int
type ErrorCode int
type Config string

const (
	Pending Status = 0
	Started Status = 1
	Running Status = 2
	Failure Status = 3
	Success Status = 4
)

// Task task model
// @Description task table
type Task struct {
	gorm.Model // embeded gorm model
	Title string
	Creator string
	Status Status
	ErrorCode ErrorCode
	Config Config `gorm:"type:json"`
	Label []string `gorm:"type:json"`  // 修改为字符串切片并使用JSON类型存储
}

type AAATask struct{
	Task
}
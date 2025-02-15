package model

import "gorm.io/gorm"

type JobResult struct {
	gorm.Model
	JobID     uint      // Foreign key for Job
	Output    string    // 主要输出结果
	Metrics   string    `gorm:"type:json"` // 结构化指标数据
	Logs      string    // 执行日志
	Artifacts []string  `gorm:"type:json"` // 输出产物路径
}

type Job struct {
	gorm.Model
	Title      string
	Creator    string
	Status     Status
	ErrorCode  ErrorCode
	Config     Config    `gorm:"type:jsonb"`
	Label      []string  `gorm:"type:jsonb"`
	Result     JobResult `gorm:"foreignKey:JobID"`
	Tasks      []Task    `gorm:"foreignKey:JobID"`
}
// Package model 定义 GORM 实体与领域枚举。
package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 统一主键、时间戳与软删除字段，并固定 JSON 命名风格。
// 不直接嵌入 gorm.Model，避免响应体里出现 ID / CreatedAt 这类大写字段。
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Status 是 Job/Task 的执行状态。
type Status int

const (
	StatusPending Status = iota
	StatusStarted
	StatusRunning
	StatusFailure
	StatusSuccess
)

// AllStatuses 返回全部合法状态，用于校验与文档。
func AllStatuses() []Status {
	return []Status{StatusPending, StatusStarted, StatusRunning, StatusFailure, StatusSuccess}
}

func (s Status) Valid() bool {
	return s >= StatusPending && s <= StatusSuccess
}

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusStarted:
		return "started"
	case StatusRunning:
		return "running"
	case StatusFailure:
		return "failure"
	case StatusSuccess:
		return "success"
	default:
		return "unknown"
	}
}

// ErrorCode 是失败原因码。
type ErrorCode int

const (
	ErrorNone ErrorCode = iota
	ErrorInvalidInput
	ErrorDependencyFailed
	ErrorTimeout
	ErrorInternal
)

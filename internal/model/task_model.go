package model

// Task 是 Job 下的最小执行单元。
type Task struct {
	BaseModel
	JobID     uint           `gorm:"index;not null" json:"job_id"`
	Title     string         `gorm:"size:255;not null" json:"title"`
	Creator   string         `gorm:"size:64;index" json:"creator"`
	Status    Status         `gorm:"not null;default:0;index" json:"status"`
	ErrorCode ErrorCode      `gorm:"not null;default:0" json:"error_code"`
	Config    map[string]any `gorm:"serializer:json" json:"config"`
	Labels    []string       `gorm:"serializer:json" json:"labels"`
}

func (Task) TableName() string { return "tasks" }

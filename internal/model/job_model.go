package model

// JobResult 保存 Job 的执行结果。
// 结构化字段使用 GORM 内建 serializer:json，列类型为普通文本，
// 因此同一份模型可在 sqlite / mysql / postgresql 上迁移，不依赖 jsonb。
type JobResult struct {
	BaseModel
	JobID     uint           `gorm:"index;not null" json:"job_id"`
	Output    string         `gorm:"type:text" json:"output"`
	Metrics   map[string]any `gorm:"serializer:json" json:"metrics"`
	Logs      string         `gorm:"type:text" json:"logs"`
	Artifacts []string       `gorm:"serializer:json" json:"artifacts"`
}

func (JobResult) TableName() string { return "job_results" }

// Job 是一组 Task 的父任务。
type Job struct {
	BaseModel
	Title     string         `gorm:"size:255;not null" json:"title"`
	Creator   string         `gorm:"size:64;index" json:"creator"`
	Status    Status         `gorm:"not null;default:0;index" json:"status"`
	ErrorCode ErrorCode      `gorm:"not null;default:0" json:"error_code"`
	Config    map[string]any `gorm:"serializer:json" json:"config"`
	Labels    []string       `gorm:"serializer:json" json:"labels"`
	Result    JobResult      `gorm:"foreignKey:JobID" json:"result"`
	Tasks     []Task         `gorm:"foreignKey:JobID" json:"tasks,omitempty"`
}

func (Job) TableName() string { return "jobs" }

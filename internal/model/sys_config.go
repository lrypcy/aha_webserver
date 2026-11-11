package model

// SysConfig 是运行时配置表（保留给将来需要动态配置的场景）。
// 注意：JWT 密钥属于启动期机密，只从环境变量读取，不存放于此表。
type SysConfig struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Key         string `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value       string `gorm:"type:text" json:"value"`
	Description string `gorm:"size:255" json:"description"`
}

func (SysConfig) TableName() string { return "sys_config" }

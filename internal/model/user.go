package model

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	// RoleUser 是普通用户角色。
	RoleUser = "user"
	// RoleAdmin 是管理员角色。
	RoleAdmin = "admin"
)

// User 是系统用户。
type User struct {
	BaseModel
	Username  string     `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Email     string     `gorm:"size:128;uniqueIndex;not null" json:"email"`
	Password  string     `gorm:"size:255;not null" json:"-"`
	Role      string     `gorm:"size:16;not null;default:user" json:"role"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	IsActive  bool       `gorm:"not null;default:true" json:"is_active"`
}

func (User) TableName() string { return "users" }

// UserInfo 是可以安全返回给客户端的用户视图。
type UserInfo struct {
	ID        uint       `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
}

// Info 返回脱敏后的用户视图。
func (u *User) Info() UserInfo {
	return UserInfo{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		LastLogin: u.LastLogin,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}

// BeforeCreate 在创建前对密码做 bcrypt 哈希。
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Role == "" {
		u.Role = RoleUser
	}
	return u.SetPassword(u.Password)
}

// SetPassword 生成 bcrypt 哈希并替换明文。
func (u *User) SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		// 注意：%w 只能包装 error，不能包装 string。
		return fmt.Errorf("hash password: %w", err)
	}
	u.Password = string(hash)
	return nil
}

// VerifyPassword 校验明文密码，绝不打印密码或哈希。
func (u *User) VerifyPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
}

// MarkLogin 记录最近登录时间。
func (u *User) MarkLogin(t time.Time) { u.LastLogin = &t }

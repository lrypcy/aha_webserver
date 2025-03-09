package model

import (
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username  string    `gorm:"size:255;uniqueIndex;not null" json:"username" validate:"required,min=3,max=50"`
	Email     string    `gorm:"size:255;uniqueIndex" json:"email" validate:"required,email"`
	Password  string    `gorm:"size:255;not null" json:"-" validate:"required,min=8"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	IsActive  bool        `gorm:"default:true" json:"is_active"`
}

func (User) TableName() string {
	return "users"
}

// 创建用户前的密码加密钩子
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// TODO: 删除BeforeSave钩子，只保留BeforeCreate，并确保在更新密码时，手动调用加密（如UpdatePassword方法已经做了加密，因此在保存时不需要再次加密）。

// 验证用户密码
func (u *User) VerifyPassword(password string) bool {
	fmt.Println(u.Password)
	fmt.Println(password)
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// 获取用户信息
func (u *User) GetUserInfo() map[string]interface{} {
	return map[string]interface{}{
		"id":        u.ID,
		"username":  u.Username,
		"email":     u.Email,
		"last_login": u.LastLogin,
		"is_active": u.IsActive,
	}
}

// 更新用户信息
func (u *User) UpdateUserInfo(info map[string]interface{}) error {
	if username, ok := info["username"].(string); ok {
		u.Username = username
	}
	if email, ok := info["email"].(string); ok {
		u.Email = email
	}
	if lastLogin, ok := info["last_login"].(*time.Time); ok {
		u.LastLogin = lastLogin
	}
	if isActive, ok := info["is_active"].(bool); ok {
		u.IsActive = isActive
	}
	return nil
}

// 更新用户密码
func (u *User) UpdatePassword(newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf( "%w, failed to update password", err.Error())
	}
	u.Password = string(hashedPassword)
	return nil
}

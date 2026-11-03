// Package setup 提供安装与运维命令用到的小工具：生成 APP_KEY、创建管理员。
package setup

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"xboard/internal/auth"
	"xboard/internal/model"
)

// NewAppKey 生成与 Laravel 一致格式的 APP_KEY（base64: + 32 字节随机数）。
func NewAppKey() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "base64:" + base64.StdEncoding.EncodeToString(b)
}

// CreateAdmin 创建管理员；邮箱已存在时提升为管理员（若给了密码则同时重置密码）。
// password 为空时随机生成并返回。
func CreateAdmin(db *gorm.DB, email, password string) (final string, created bool, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || len(email) < 3 {
		return "", false, errors.New("请输入有效的邮箱地址")
	}
	generated := false
	if password == "" {
		password, generated = auth.RandHex(8), true
	}
	if len(password) < 8 {
		return "", false, errors.New("管理员密码长度最小为8位字符")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", false, err
	}
	now := time.Now().Unix()

	var existing model.User
	if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
		upd := map[string]any{"is_admin": true, "updated_at": now}
		if !generated {
			upd["password"] = hash
		}
		if err := db.Model(&model.User{}).Where("id = ?", existing.ID).Updates(upd).Error; err != nil {
			return "", false, err
		}
		if generated {
			return "", false, nil // 已存在且未指定密码：只提权，不改密码
		}
		return password, false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, err
	}

	u := model.User{
		Email: email, Password: hash, UUID: uuid.NewString(), Token: auth.RandHex(16),
		IsAdmin: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&u).Error; err != nil {
		return "", false, fmt.Errorf("创建管理员失败：%w", err)
	}
	return password, true, nil
}

// HasAdmin 判断库里是否已经有管理员。
func HasAdmin(db *gorm.DB) bool {
	var n int64
	db.Model(&model.User{}).Where("is_admin = ?", true).Count(&n)
	return n > 0
}

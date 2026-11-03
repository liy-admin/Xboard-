package auth

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"xboard/internal/model"
)

// VerifyPassword 对应 Helper::multiPasswordVerify。
func VerifyPassword(u *model.User, pw string) bool {
	algo, salt := "", ""
	if u.PasswordAlgo != nil {
		algo = *u.PasswordAlgo
	}
	if u.PasswordSalt != nil {
		salt = *u.PasswordSalt
	}
	md := func(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }
	sh := func(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
	switch algo {
	case "md5":
		return md(pw) == u.Password
	case "sha256":
		return sh(pw) == u.Password
	case "md5salt":
		return md(pw+salt) == u.Password
	case "sha256salt":
		return sh(pw+salt) == u.Password
	}
	hash := u.Password
	if strings.HasPrefix(hash, "$2y$") { // PHP 的 $2y$ 与 $2a$ 等价
		hash = "$2a$" + hash[4:]
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	return string(b), err
}

func RandHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Issue 写入 Sanctum 兼容 token，返回客户端使用的 "Bearer xxx"。
func Issue(db *gorm.DB, uid int64, name string) (string, error) {
	plain := RandHex(20)
	sum := sha256.Sum256([]byte(plain))
	now := time.Now().Format("2006-01-02 15:04:05")
	exp := time.Now().AddDate(1, 0, 0).Format("2006-01-02 15:04:05")
	abil := `["*"]`
	t := model.PersonalAccessToken{
		TokenableType: "App\\Models\\User", TokenableID: uid, Name: name,
		Token: hex.EncodeToString(sum[:]), Abilities: &abil,
		ExpiresAt: &exp, CreatedAt: &now, UpdatedAt: &now,
	}
	if err := db.Create(&t).Error; err != nil {
		return "", err
	}
	return "Bearer " + plain, nil
}

// UserFromBearer 校验 Sanctum token 并取出用户。
func UserFromBearer(db *gorm.DB, header string) *model.User {
	plain := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if plain == "" {
		return nil
	}
	if i := strings.Index(plain, "|"); i >= 0 {
		plain = plain[i+1:]
	}
	sum := sha256.Sum256([]byte(plain))
	var t model.PersonalAccessToken
	if err := db.Where("token = ?", hex.EncodeToString(sum[:])).First(&t).Error; err != nil {
		return nil
	}
	if t.ExpiresAt != nil && *t.ExpiresAt != "" {
		e, ok := ParseDBTime(*t.ExpiresAt)
		if !ok || e.Before(time.Now()) {
			return nil // 无法解析的到期时间同样视为失效，避免过期令牌因格式问题而永久有效
		}
	}
	var u model.User
	if err := db.First(&u, t.TokenableID).Error; err != nil {
		return nil
	}
	return &u
}

func ConstEq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

// ParseDBTime 解析从数据库扫描出来的时间字符串：驱动开启 parseTime 时是 RFC3339，
// 否则是 "2006-01-02 15:04:05"（本地时区）。
func ParseDBTime(s string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, true
	}
	return time.Time{}, false
}

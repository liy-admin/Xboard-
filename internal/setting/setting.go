package setting

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"xboard/internal/model"
)

// Service 等价于 admin_setting()：整表读入内存，30s 刷新。
type Service struct {
	db   *gorm.DB
	mu   sync.RWMutex
	data map[string]string
	at   time.Time
}

func New(db *gorm.DB) *Service { return &Service{db: db, data: map[string]string{}} }

// NewFromMap 创建一个不连数据库、配置固定为 m 的服务，供测试使用。
func NewFromMap(m map[string]string) *Service {
	data := make(map[string]string, len(m))
	for k, v := range m {
		data[k] = v
	}
	return &Service{data: data}
}

func (s *Service) load() {
	if s.db == nil {
		return
	}
	s.mu.RLock()
	fresh := time.Since(s.at) < 30*time.Second
	s.mu.RUnlock()
	if fresh {
		return
	}
	var rows []model.Setting
	if err := s.db.Find(&rows).Error; err != nil {
		return
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Value != nil {
			m[r.Name] = *r.Value
		}
	}
	s.mu.Lock()
	s.data, s.at = m, time.Now()
	s.mu.Unlock()
}

func (s *Service) Get(name, def string) string {
	s.load()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.data[name]; ok {
		return v
	}
	return def
}

// Int 对应 (int) admin_setting(name, def)：配置存在时按整数解析（非数字为 0），缺失时用 def。
func (s *Service) Int(name string, def int) int {
	if !s.Has(name) {
		return def
	}
	if n, err := strconv.Atoi(strings.TrimSpace(s.Get(name, ""))); err == nil {
		return n
	}
	return 0
}

// Bool 解析布尔型配置，"1"/"true" 为真，支持 PHP 风格的 "0"/"" 为假。
// Bool 对应 (bool) admin_setting(name, def)：PHP 保存 false 时写入空串，空串是有效值（为假），只有缺失才用 def。
func (s *Service) Bool(name string, def bool) bool {
	if !s.Has(name) {
		return def
	}
	v := s.Get(name, "")
	return v != "" && v != "0" && v != "false"
}

// Float 解析浮点型配置（如奖励比例），解析失败返回 def。
func (s *Service) Float(name string, def float64) float64 {
	if f, err := strconv.ParseFloat(s.Get(name, ""), 64); err == nil {
		return f
	}
	return def
}

// Has 判断配置项是否存在（即使值为空），对应 admin_setting 不带默认值时区分 null 与缺失。
func (s *Service) Has(name string) bool {
	s.load()
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[name]
	return ok
}

// Invalidate 立即丢弃内存缓存，下次读取时重新加载（后台保存配置后调用）。
func (s *Service) Invalidate() {
	s.mu.Lock()
	s.at = time.Time{}
	s.mu.Unlock()
}

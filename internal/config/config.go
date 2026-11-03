package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	// 内嵌时区数据库：精简系统（容器 / 最小化镜像）没有 /usr/share/zoneinfo 也能加载 Asia/Shanghai。
	_ "time/tzdata"
)

type Config struct {
	Listen    string
	DBHost    string
	DBPort    string
	DBName    string
	DBUser    string
	DBPass    string
	RedisAddr string
	RedisPass string
	RedisDB   int
	AppKey    string
	Timezone  string
	// TrustedProxies 为可信反向代理网段 / 地址；决定能否信任 X-Forwarded-For 取客户端 IP。
	TrustedProxies []string
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" && v != "null" {
		return v
	}
	return def
}

// LoadDotEnv 读取 .env 文件（XBOARD_ENV_FILE 指定路径，缺省为当前目录的 .env）。
// 已存在的真实环境变量优先，不会被文件覆盖，与 Laravel 的 Dotenv 行为一致。
// 返回读取的文件路径；文件不存在时返回空串。
func LoadDotEnv() string {
	path := os.Getenv("XBOARD_ENV_FILE")
	if path == "" {
		path = ".env"
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			v = v[1 : len(v)-1]
		} else if i := strings.Index(v, " #"); i >= 0 { // 行尾注释
			v = strings.TrimSpace(v[:i])
		}
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
	return path
}

// Load 读取与原 Laravel .env 相同的环境变量名，可原地替换部署。
func Load() *Config {
	LoadDotEnv()
	db, _ := strconv.Atoi(get("REDIS_DB", "0"))
	c := &Config{
		Listen:    get("LISTEN", ":7001"),
		DBHost:    get("DB_HOST", "127.0.0.1"),
		DBPort:    get("DB_PORT", "3306"),
		DBName:    get("DB_DATABASE", "xboard"),
		DBUser:    get("DB_USERNAME", "root"),
		DBPass:    get("DB_PASSWORD", ""),
		RedisAddr: get("REDIS_HOST", "127.0.0.1") + ":" + get("REDIS_PORT", "6379"),
		RedisPass: get("REDIS_PASSWORD", ""),
		RedisDB:   db,
		AppKey:    get("APP_KEY", ""),
		Timezone:  get("APP_TIMEZONE", "Asia/Shanghai"),
	}
	// REDIS_HOST 为 unix socket（Docker 内置 Redis 的 /data/redis.sock）时不拼端口。
	if strings.HasPrefix(c.RedisAddr, "/") {
		c.RedisAddr = strings.TrimSuffix(c.RedisAddr, ":"+get("REDIS_PORT", "6379"))
		c.RedisAddr = strings.TrimSuffix(c.RedisAddr, ":0")
	}
	for _, p := range strings.Split(get("TRUSTED_PROXIES", "127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			c.TrustedProxies = append(c.TrustedProxies, p)
		}
	}
	// 与 PHP（config/app.php 的 timezone）一致：日界、定时任务、统计都按应用时区计算。
	if loc, err := time.LoadLocation(c.Timezone); err == nil {
		time.Local = loc
	}
	return c
}

package store

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"xboard/internal/config"
)

type Store struct {
	DB    *gorm.DB
	Redis *redis.Client
}

// DSN 返回连接 MySQL / MariaDB 的 DSN；name 为空时不指定库（用于创建库）。
func DSN(c *config.Config, name string, multiStatements bool) string {
	loc := "Local"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, name, loc)
	if multiStatements {
		dsn += "&multiStatements=true"
	}
	return dsn
}

func Open(c *config.Config) (*Store, error) {
	// 缺省只记录慢查询与错误；「记录不存在」是正常分支（大量 First 查询），不应刷屏。
	level := logger.Warn
	if os.Getenv("XBOARD_SQL_DEBUG") == "1" {
		level = logger.Info
	}
	gl := logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
		SlowThreshold:             500 * time.Millisecond,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
	})
	db, err := gorm.Open(mysql.Open(DSN(c, c.DBName, false)), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         gl,
	})
	if err != nil {
		return nil, err
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(50)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}
	opt := &redis.Options{Addr: c.RedisAddr, Password: c.RedisPass, DB: c.RedisDB}
	if strings.HasPrefix(c.RedisAddr, "/") {
		opt.Network = "unix"
	}
	return &Store{DB: db, Redis: redis.NewClient(opt)}, nil
}

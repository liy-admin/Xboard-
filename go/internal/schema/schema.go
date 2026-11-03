// Package schema 内置数据库结构，使二进制可以独立建表（xboard migrate / install），不依赖 PHP 的 artisan。
//
// schema.sql / seed.sql 由 PHP 自带迁移（database/migrations）在全新库上执行后导出，
// 因此与 PHP 版表结构一致；seed.sql 只包含迁移自带的数据（migrations 记录与订阅模板）。
// 重新生成：scripts/wsl/dump_schema.sh（需要 PHP 与 MariaDB）。
package schema

import (
	"database/sql"
	"embed"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"xboard/internal/config"
	"xboard/internal/store"
)

//go:embed schema.sql seed.sql
var files embed.FS

// Result 描述一次迁移的结果。
type Result struct {
	Fresh   bool // 迁移前库里没有业务表（v2_user 不存在）
	Tables  int  // 迁移后的表数量
	Created int  // 本次新建的表数量
}

// EnsureDatabase 在库不存在时创建它（utf8mb4）。
func EnsureDatabase(c *config.Config) error {
	db, err := sql.Open("mysql", store.DSN(c, "", false))
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败：%w", err)
	}
	name := strings.ReplaceAll(c.DBName, "`", "")
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	return err
}

var jsonColumn = regexp.MustCompile("longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin( DEFAULT NULL| NOT NULL)?( COMMENT '[^']*')? CHECK \\(json_valid\\(`[^`]+`\\)\\)")

func countTables(db *sql.DB, name string) (int, error) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", name).Scan(&n)
	return n, err
}

func tableExists(db *sql.DB, schemaName, table string) bool {
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", schemaName, table).Scan(&n)
	return n > 0
}

func columnExists(db *sql.DB, schemaName, table, column string) bool {
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?",
		schemaName, table, column).Scan(&n)
	return n > 0
}

// addedColumn 是后加到已有表上的列。CREATE TABLE IF NOT EXISTS 不会改动老库里已存在的表，
// 这些列在老库上可能缺失，需要单独补齐。
type addedColumn struct {
	Table, Column string
	DDL           string   // ALTER TABLE 子句
	Backfill      []string // 列刚补上时执行一次的数据修正
}

var addedColumns = []addedColumn{
	{
		Table: "v2_user", Column: "device_id",
		DDL: "ADD COLUMN `device_id` varchar(128) DEFAULT NULL, ADD KEY `v2_user_device_id_index` (`device_id`)",
	},
	{
		// 设备是否领过试用的唯一判据；缺这列时注册/设备体验都会把试用当成"从未领过"反复发放。
		Table: "v2_device_binding", Column: "claimed_trial",
		DDL: "ADD COLUMN `claimed_trial` tinyint(1) NOT NULL DEFAULT 0 AFTER `type`",
		// 与 PHP 迁移 add_claimed_trial_to_device_binding 一致：体验影子账号来源的设备视为已领取。
		Backfill: []string{
			"UPDATE `v2_device_binding` SET `claimed_trial` = 1 WHERE `type` = 'trial'",
			"UPDATE `v2_device_binding` b SET b.`claimed_trial` = 1 WHERE EXISTS (SELECT 1 FROM `v2_user` u WHERE u.`device_id` = b.`device_id` AND u.`email` LIKE '%@device.local')",
		},
	},
}

// ensureColumns 补齐老库上缺失的后加列（幂等：列已存在则跳过）。
func ensureColumns(db *sql.DB, schemaName string) error {
	for _, col := range addedColumns {
		if !tableExists(db, schemaName, col.Table) || columnExists(db, schemaName, col.Table, col.Column) {
			continue
		}
		if _, err := db.Exec("ALTER TABLE `" + col.Table + "` " + col.DDL); err != nil {
			return fmt.Errorf("补齐字段 %s.%s 失败：%w", col.Table, col.Column, err)
		}
		for _, stmt := range col.Backfill {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("回填 %s.%s 失败：%w", col.Table, col.Column, err)
			}
		}
	}
	return nil
}

// Migrate 幂等地建表：已存在的表不动（CREATE TABLE IF NOT EXISTS）；仅在全新库上写入种子数据。
// 已存在的表只会补齐 addedColumns 里缺失的列（见 ensureColumns），不改动其他结构与数据。
func Migrate(c *config.Config) (Result, error) {
	var res Result
	if err := EnsureDatabase(c); err != nil {
		return res, err
	}
	db, err := sql.Open("mysql", store.DSN(c, c.DBName, true))
	if err != nil {
		return res, err
	}
	defer db.Close()

	before, err := countTables(db, c.DBName)
	if err != nil {
		return res, err
	}
	res.Fresh = !tableExists(db, c.DBName, "v2_user")

	ddl, err := files.ReadFile("schema.sql")
	if err != nil {
		return res, err
	}
	script := string(ddl)
	// MariaDB 把 JSON 导出为带 CHECK 的 longtext；真正的 MySQL 上还原为原生 JSON 类型。
	var version string
	_ = db.QueryRow("SELECT VERSION()").Scan(&version)
	if !strings.Contains(strings.ToLower(version), "mariadb") {
		script = jsonColumn.ReplaceAllString(script, "json$1$2")
	}
	// 导出的建表语句按表名排序，外键引用的表可能还没创建：整批执行期间关闭外键检查（同一连接内生效）。
	script = "SET FOREIGN_KEY_CHECKS=0;\n" + script + "\nSET FOREIGN_KEY_CHECKS=1;"
	if _, err := db.Exec(script); err != nil {
		return res, fmt.Errorf("建表失败：%w", err)
	}

	if err := ensureColumns(db, c.DBName); err != nil {
		return res, err
	}

	if res.Fresh {
		seed, err := files.ReadFile("seed.sql")
		if err != nil {
			return res, err
		}
		stmt := "SET FOREIGN_KEY_CHECKS=0;\n" + strings.ReplaceAll(string(seed), "INSERT INTO", "INSERT IGNORE INTO") + "\nSET FOREIGN_KEY_CHECKS=1;"
		if _, err := db.Exec(stmt); err != nil {
			return res, fmt.Errorf("写入种子数据失败：%w", err)
		}
	}
	after, err := countTables(db, c.DBName)
	if err != nil {
		return res, err
	}
	res.Tables, res.Created = after, after-before
	return res, nil
}

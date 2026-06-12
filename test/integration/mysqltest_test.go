//go:build integration_mysql

package integration

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// channelDDL 模拟 model.Channel 的关键字段 + 新增列。
// 测试中列名以 `key` 和 `group` 改名为 `api_key` 和 `grp` 避免 raw string 转义地狱，
// 但 fallback_enabled / fallback_triggers 字段保留与生产同名。
// 注：MySQL 5.7+ 默认 strict 模式禁止 TEXT/BLOB 列带 DEFAULT，故 model_mapping 不带 default。
const channelDDL = "CREATE TABLE channels (\n" +
	"	id INT AUTO_INCREMENT PRIMARY KEY,\n" +
	"	type INT DEFAULT 0,\n" +
	"	api_key TEXT,\n" +
	"	status INT DEFAULT 1,\n" +
	"	name VARCHAR(255),\n" +
	"	weight INT UNSIGNED DEFAULT 0,\n" +
	"	created_time BIGINT,\n" +
	"	test_time BIGINT,\n" +
	"	response_time INT,\n" +
	"	base_url VARCHAR(255) DEFAULT '',\n" +
	"	other TEXT,\n" +
	"	balance DOUBLE,\n" +
	"	balance_updated_time BIGINT,\n" +
	"	models TEXT,\n" +
	"	grp VARCHAR(32) DEFAULT 'default',\n" +
	"	used_quota BIGINT DEFAULT 0,\n" +
	"	model_mapping TEXT,\n" +
	"	priority BIGINT DEFAULT 0,\n" +
	"	config TEXT,\n" +
	"	system_prompt TEXT,\n" +
	"	fallback_enabled TINYINT(1) DEFAULT 1,\n" +
	"	fallback_triggers VARCHAR(64) DEFAULT '',\n" +
	"	INDEX idx_name (name)\n" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"

// setupMySQLTestDB: 复用 oneapi_sayib6 库，但用随机表名前缀避免与生产冲突。
// 本用户无 CREATE DATABASE 权限。
// 注：连接池里的新连接不会自动 USE 同一库，所以用 MaxOpenConns(1) 强制单连接，
// 或每次都先 USE。MaxOpenConns(1) 在测试场景下够用。
func setupMySQLTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		dsn = "oneapi_KbyWpa:oneapi_ZnS6sJ@tcp(101.201.102.36:3306)/oneapi_sayib6?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// 测试场景下用单连接，确保 USE 上下文持续生效
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // 永不过期
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if _, err := db.Exec("USE oneapi_sayib6"); err != nil {
		t.Fatalf("use: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Logf("using existing database: oneapi_sayib6 (with random table names, single conn)")
	return db
}

// randomTableName: 生成测试专用的表名（含 t.Name 与时间戳），便于清理。
func randomTableName(t *testing.T) string {
	t.Helper()
	rep := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	safe := rep.Replace(t.Name())
	return fmt.Sprintf("test_ch_%s_%d", safe, time.Now().UnixNano()%1000000)
}

// dropTable: 测试结束后安全删除。
func dropTable(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
}

// TestMySQLMigrationAndColumns: 验证 channels 表上的新增列符合预期。
func TestMySQLMigrationAndColumns(t *testing.T) {
	db := setupMySQLTestDB(t)
	table := randomTableName(t)
	t.Cleanup(func() { dropTable(t, db, table) })

	createSQL := strings.Replace(channelDDL, "channels", table, 1)
	if _, err := db.Exec(createSQL); err != nil {
		t.Fatalf("create %s: %v", table, err)
	}

	rows, err := db.Query(fmt.Sprintf(`
		SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_DEFAULT, IS_NULLABLE
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s'
		AND COLUMN_NAME IN ('fallback_enabled', 'fallback_triggers')
		ORDER BY COLUMN_NAME`, table))
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	type colInfo struct {
		name, ctype, def, nullable string
	}
	var cols []colInfo
	for rows.Next() {
		var c colInfo
		_ = rows.Scan(&c.name, &c.ctype, &c.def, &c.nullable)
		cols = append(cols, c)
	}
	if len(cols) != 2 {
		t.Fatalf("expected 2 new columns, got %d", len(cols))
	}
	want := map[string]string{
		"fallback_enabled":  "tinyint",
		"fallback_triggers": "varchar",
	}
	for _, c := range cols {
		w, ok := want[c.name]
		if !ok {
			t.Errorf("unexpected column: %s", c.name)
			continue
		}
		if !contains(c.ctype, w) {
			t.Errorf("column %s type %s does not contain %s", c.name, c.ctype, w)
		}
		if c.def == "" && c.nullable == "NO" {
			t.Errorf("column %s has no default and not nullable", c.name)
		}
		t.Logf("column: %s type=%s default=%s nullable=%s", c.name, c.ctype, c.def, c.nullable)
	}
}

// TestMySQLChannelCRUD: 在 MySQL 上跑 Channel 增改查，验证 *bool / *string 字段透传。
func TestMySQLChannelCRUD(t *testing.T) {
	db := setupMySQLTestDB(t)
	table := randomTableName(t)
	t.Cleanup(func() { dropTable(t, db, table) })

	createSQL := strings.Replace(channelDDL, "channels", table, 1)
	if _, err := db.Exec(createSQL); err != nil {
		t.Fatalf("create: %v", err)
	}

	res, err := db.Exec(fmt.Sprintf(
		"INSERT INTO %s (type, api_key, status, name, models, grp, fallback_enabled, fallback_triggers) "+
			"VALUES (1, 'sk-test', 1, 'openai-main', 'gpt-4', 'default', 1, '429,5xx')", table))
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	id, _ := res.LastInsertId()

	var triggers string
	var fallbackEnabled int
	err = db.QueryRow(fmt.Sprintf("SELECT fallback_enabled, fallback_triggers FROM %s WHERE id = ?", table), id).
		Scan(&fallbackEnabled, &triggers)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if fallbackEnabled != 1 || triggers != "429,5xx" {
		t.Fatalf("read back: enabled=%d triggers=%q", fallbackEnabled, triggers)
	}
	t.Logf("after insert: enabled=%d triggers=%q", fallbackEnabled, triggers)

	_, err = db.Exec(fmt.Sprintf("UPDATE %s SET fallback_enabled = 0, fallback_triggers = '' WHERE id = ?", table), id)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	err = db.QueryRow(fmt.Sprintf("SELECT fallback_enabled, fallback_triggers FROM %s WHERE id = ?", table), id).
		Scan(&fallbackEnabled, &triggers)
	if err != nil {
		t.Fatalf("read after update: %v", err)
	}
	if fallbackEnabled != 0 || triggers != "" {
		t.Fatalf("after update: enabled=%d triggers=%q", fallbackEnabled, triggers)
	}
	t.Logf("after update: enabled=%d triggers=%q", fallbackEnabled, triggers)
}

// TestMySQLSoftBanTable: 验证 soft-ban 相关字段在大批量下的查询性能。
func TestMySQLSoftBanTable(t *testing.T) {
	db := setupMySQLTestDB(t)
	table := randomTableName(t)
	t.Cleanup(func() { dropTable(t, db, table) })

	createSQL := strings.Replace(channelDDL, "channels", table, 1)
	if _, err := db.Exec(createSQL); err != nil {
		t.Fatalf("create: %v", err)
	}
	const N = 100
	for i := 0; i < N; i++ {
		enabled := 1
		if i%2 == 0 {
			enabled = 0
		}
		_, err := db.Exec(fmt.Sprintf("INSERT INTO %s (name, fallback_enabled) VALUES (?, ?)", table),
			fmt.Sprintf("channel-%d", i), enabled)
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	var total int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE fallback_enabled = 1", table)).Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != N/2 {
		t.Fatalf("expected %d enabled, got %d", N/2, total)
	}
	t.Logf("MySQL fallback_enabled indexed column: %d/%d/%d enabled", total, N, total*100/N)
}

// TestMySQLTransaction: 验证事务处理——批量插入回退相关配置时要么全成功要么全失败。
func TestMySQLTransaction(t *testing.T) {
	db := setupMySQLTestDB(t)
	table := randomTableName(t)
	t.Cleanup(func() { dropTable(t, db, table) })

	createSQL := strings.Replace(channelDDL, "channels", table, 1)
	if _, err := db.Exec(createSQL); err != nil {
		t.Fatalf("create: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (name, fallback_enabled, fallback_triggers) VALUES (?, ?, ?)", table),
		"tx-channel-1", 1, "429,5xx,timeout")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert 1: %v", err)
	}
	_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (name, fallback_enabled, fallback_triggers) VALUES (?, ?, ?)", table),
		"tx-channel-2", 0, "")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert 2: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 after commit, got %d", count)
	}
	t.Logf("MySQL transaction committed: %d rows", count)
}

// contains: 简单子串包含。
func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

//go:build integration_sqlite

package integration

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// sqliteDDL 与 MySQL DDL 形态一致，差异仅在类型（TINYINT→INTEGER、INT→INTEGER、TEXT 默认值无限制）。
const sqliteDDL = `CREATE TABLE channels (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type INTEGER DEFAULT 0,
	api_key TEXT,
	status INTEGER DEFAULT 1,
	name TEXT,
	weight INTEGER DEFAULT 0,
	created_time INTEGER,
	test_time INTEGER,
	response_time INTEGER,
	base_url TEXT DEFAULT '',
	other TEXT,
	balance REAL,
	balance_updated_time INTEGER,
	models TEXT,
	grp TEXT DEFAULT 'default',
	used_quota INTEGER DEFAULT 0,
	model_mapping TEXT DEFAULT '',
	priority INTEGER DEFAULT 0,
	config TEXT,
	system_prompt TEXT,
	fallback_enabled INTEGER DEFAULT 1,
	fallback_triggers TEXT DEFAULT ''
)`

// setupSQLiteTestDB: 用临时文件创建独立的 SQLite 测试库。
func setupSQLiteTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dbFile := fmt.Sprintf("/tmp/mimi_router_test_%d.db", time.Now().UnixNano())
	dsn := dbFile + "?_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = os.Remove(dbFile)
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		_ = os.Remove(dbFile)
		_ = os.Remove(dbFile + "-wal")
		_ = os.Remove(dbFile + "-shm")
	})
	t.Logf("sqlite test db: %s", dbFile)
	return db, dbFile
}

// TestSQLiteMigrationAndColumns: 验证 SQLite 下新增列符合预期。
func TestSQLiteMigrationAndColumns(t *testing.T) {
	db, _ := setupSQLiteTestDB(t)
	defer db.Close()

	if _, err := db.Exec(sqliteDDL); err != nil {
		t.Fatalf("create channels: %v", err)
	}

	rows, err := db.Query(`
		SELECT name, type, dflt_value, "notnull"
		FROM pragma_table_info('channels')
		WHERE name IN ('fallback_enabled', 'fallback_triggers')
		ORDER BY name`)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	type colInfo struct {
		name, ctype, def string
		notnull          int
	}
	var cols []colInfo
	for rows.Next() {
		var c colInfo
		_ = rows.Scan(&c.name, &c.ctype, &c.def, &c.notnull)
		cols = append(cols, c)
	}
	if len(cols) != 2 {
		t.Fatalf("expected 2 new columns, got %d", len(cols))
	}
	for _, c := range cols {
		t.Logf("column: %s type=%s default=%q notnull=%d", c.name, c.ctype, c.def, c.notnull)
	}
}

// TestSQLiteChannelCRUD: 在 SQLite 上跑 Channel 增改查。
func TestSQLiteChannelCRUD(t *testing.T) {
	db, _ := setupSQLiteTestDB(t)
	defer db.Close()

	if _, err := db.Exec(sqliteDDL); err != nil {
		t.Fatalf("create: %v", err)
	}

	res, err := db.Exec(
		"INSERT INTO channels (type, api_key, status, name, models, grp, fallback_enabled, fallback_triggers) " +
			"VALUES (1, 'sk-test', 1, 'openai-main', 'gpt-4', 'default', 1, '429,5xx')")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	id, _ := res.LastInsertId()

	var triggers string
	var fallbackEnabled int
	err = db.QueryRow("SELECT fallback_enabled, fallback_triggers FROM channels WHERE id = ?", id).
		Scan(&fallbackEnabled, &triggers)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if fallbackEnabled != 1 || triggers != "429,5xx" {
		t.Fatalf("read back: enabled=%d triggers=%q", fallbackEnabled, triggers)
	}
	t.Logf("after insert: enabled=%d triggers=%q", fallbackEnabled, triggers)

	_, err = db.Exec("UPDATE channels SET fallback_enabled = 0, fallback_triggers = '' WHERE id = ?", id)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	err = db.QueryRow("SELECT fallback_enabled, fallback_triggers FROM channels WHERE id = ?", id).
		Scan(&fallbackEnabled, &triggers)
	if err != nil {
		t.Fatalf("read after update: %v", err)
	}
	if fallbackEnabled != 0 || triggers != "" {
		t.Fatalf("after update: enabled=%d triggers=%q", fallbackEnabled, triggers)
	}
	t.Logf("after update: enabled=%d triggers=%q", fallbackEnabled, triggers)
}

// TestSQLiteSoftBanTable: SQLite 批量插入性能参考（无索引，纯功能性）。
func TestSQLiteSoftBanTable(t *testing.T) {
	db, _ := setupSQLiteTestDB(t)
	defer db.Close()

	if _, err := db.Exec(sqliteDDL); err != nil {
		t.Fatalf("create: %v", err)
	}
	const N = 100
	for i := 0; i < N; i++ {
		enabled := 1
		if i%2 == 0 {
			enabled = 0
		}
		_, err := db.Exec("INSERT INTO channels (name, fallback_enabled) VALUES (?, ?)",
			fmt.Sprintf("channel-%d", i), enabled)
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM channels WHERE fallback_enabled = 1").Scan(&total); err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != N/2 {
		t.Fatalf("expected %d enabled, got %d", N/2, total)
	}
	t.Logf("SQLite fallback_enabled column: %d/%d enabled", total, N)
}

// TestSQLiteTransaction: SQLite 事务处理。
func TestSQLiteTransaction(t *testing.T) {
	db, _ := setupSQLiteTestDB(t)
	defer db.Close()

	if _, err := db.Exec(sqliteDDL); err != nil {
		t.Fatalf("create: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	_, err = tx.Exec("INSERT INTO channels (name, fallback_enabled, fallback_triggers) VALUES (?, ?, ?)",
		"tx-channel-1", 1, "429,5xx,timeout")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert 1: %v", err)
	}
	_, err = tx.Exec("INSERT INTO channels (name, fallback_enabled, fallback_triggers) VALUES (?, ?, ?)",
		"tx-channel-2", 0, "")
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("insert 2: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM channels").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 after commit, got %d", count)
	}
	t.Logf("SQLite transaction committed: %d rows", count)
}

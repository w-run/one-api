//go:build integration_sqlite

package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	dbmodel "github.com/w-run/mimi-router/model"
)

// TestPerfFallbackPickAtScale: 1000 渠道规模下回退挑选的查询延迟。
// 重点验证 (a) DB 全表查询 latency, (b) 软禁状态查询 latency。
func TestPerfFallbackPickAtScale(t *testing.T) {
	db, _ := setupSQLiteTestDB(t)

	if _, err := db.Exec(sqliteDDL); err != nil {
		t.Fatalf("create: %v", err)
	}
	const N = 1000
	tx, _ := db.Begin()
	for i := 0; i < N; i++ {
		_, err := tx.Exec(
			"INSERT INTO channels (name, models, grp, priority, fallback_enabled) VALUES (?, ?, ?, ?, ?)",
			fmt.Sprintf("channel-%d", i+1), "gpt-4", "default", int64(i%10), 1)
		if err != nil {
			tx.Rollback()
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// 软禁前 100 个 channel, 模拟限流场景
	for i := 0; i < 100; i++ {
		dbmodel.SoftBanChannel(80000+i, time.Now().Unix()+60)
	}
	defer func() {
		for i := 0; i < 100; i++ {
			dbmodel.UnsoftBanChannel(80000 + i)
		}
	}()

	// 性能测试 1: 1000 渠道全表查询 (priority + fallback_enabled 过滤) 的延迟
	const queryIters = 1000
	start := time.Now()
	for i := 0; i < queryIters; i++ {
		rows, err := db.Query(
			"SELECT id, name, priority, fallback_enabled FROM channels WHERE grp = ? AND fallback_enabled = 1 ORDER BY priority ASC, id ASC",
			"default")
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		// 简单遍历 100 行模拟业务侧过滤
		count := 0
		for rows.Next() {
			count++
		}
		rows.Close()
		_ = count
	}
	dur := time.Since(start)
	avg := dur / queryIters
	t.Logf("channel_query[1000 channels]: %d iters in %s (avg %s/op)", queryIters, dur, avg)
	if avg > 5*time.Millisecond {
		t.Errorf("channel_query too slow: avg %s > 5ms", avg)
	}

	// 性能测试 2: 10000 次 IsChannelSoftBanned 调用
	const banIters = 10000
	start = time.Now()
	for i := 0; i < banIters; i++ {
		_ = dbmodel.IsChannelSoftBanned(80000 + (i % 100))
	}
	dur = time.Since(start)
	avg = dur / banIters
	t.Logf("IsChannelSoftBanned: %d iters in %s (avg %s/op)", banIters, dur, avg)
	if avg > 100*time.Microsecond {
		t.Errorf("IsChannelSoftBanned too slow: avg %s > 100µs", avg)
	}
}

// TestPerfSoftBanBanWrite: 10000 次软禁写入的吞吐。
func TestPerfSoftBanBanWrite(t *testing.T) {
	const iters = 10000
	// 清理可能残留
	for i := 0; i < iters; i++ {
		dbmodel.UnsoftBanChannel(90000 + i)
	}

	start := time.Now()
	for i := 0; i < iters; i++ {
		dbmodel.SoftBanChannel(90000+i, time.Now().Unix()+60)
	}
	dur := time.Since(start)
	avg := dur / iters
	t.Logf("SoftBanChannel: %d iters in %s (avg %s/op)", iters, dur, avg)
	defer func() {
		for i := 0; i < iters; i++ {
			dbmodel.UnsoftBanChannel(90000 + i)
		}
	}()
	if avg > 50*time.Microsecond {
		t.Errorf("SoftBanChannel too slow: avg %s > 50µs", avg)
	}
}

// TestPerfSoftBanConcurrent: 并发软禁读写的吞吐与安全性。
func TestPerfSoftBanConcurrent(t *testing.T) {
	const N = 2000
	var wg sync.WaitGroup
	wg.Add(N * 2)

	start := time.Now()
	for i := 0; i < N; i++ {
		go func(id int) {
			defer wg.Done()
			dbmodel.SoftBanChannel(id, time.Now().Unix()+60)
		}(i)
	}
	for i := 0; i < N; i++ {
		go func(id int) {
			defer wg.Done()
			_ = dbmodel.IsChannelSoftBanned(id)
		}(i)
	}
	wg.Wait()
	dur := time.Since(start)
	t.Logf("concurrent soft-ban %d goroutines in %s (%.0f ops/s)",
		N*2, dur, float64(N*2)/dur.Seconds())

	// 清理
	for i := 0; i < N; i++ {
		dbmodel.UnsoftBanChannel(i)
	}
}

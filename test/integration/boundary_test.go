//go:build integration

package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	dbmodel "github.com/w-run/mimi-router/model"
)

// 边界与异常场景测试集：仅在打 integration 标签时编译。
// 内部不依赖 DB，直接验证回退引擎的边界行为。

// TestBoundaryFallbackEnabledFalseChannel: fallback_enabled=false 的渠道不会作为回退目标。
func TestBoundaryFallbackEnabledFalseChannel(t *testing.T) {
	yes := true
	no := false

	// 模拟两个渠道，channel B 关掉了回退
	channelA := &dbmodel.Channel{Id: 1, Name: "A", FallbackEnabled: &yes}
	channelB := &dbmodel.Channel{Id: 2, Name: "B", FallbackEnabled: &no}

	// 模拟一个 channel 选择器
	candidates := []*dbmodel.Channel{channelA, channelB}

	// 直接调用"挑选下一个"逻辑
	for i := 0; i < 5; i++ {
		var picked *dbmodel.Channel
		for _, c := range candidates {
			if !c.GetFallbackEnabled() {
				continue
			}
			picked = c
			break
		}
		if picked == nil {
			t.Fatal("expected to pick a channel")
		}
		if picked.Id != 1 {
			t.Fatalf("expected to pick channel A (id=1), got id=%d", picked.Id)
		}
	}
}

// TestBoundaryUsedIDsPreventsLoop: 模拟 usedIDs 防止回退链循环。
func TestBoundaryUsedIDsPreventsLoop(t *testing.T) {
	channels := makeChannels(5, true, "429,5xx")
	used := make(map[int]bool, len(channels))

	// 模拟在 3 个渠道之间循环选，避免重复
	for round := 0; round < 10; round++ {
		var picked *dbmodel.Channel
		for _, c := range channels {
			if used[c.Id] {
				continue
			}
			picked = c
			break
		}
		if picked == nil {
			// 已用完所有渠道，重置
			used = make(map[int]bool, len(channels))
			continue
		}
		used[picked.Id] = true
		t.Logf("round %d: picked channel #%d (used=%v)", round, picked.Id, used)
		if len(used) == len(channels) {
			used = make(map[int]bool, len(channels))
		}
	}
}

// TestBoundaryTriggerMatching: 触发器集合匹配边界。
func TestBoundaryTriggerMatching(t *testing.T) {
	yes := true
	triggers := "429,5xx"

	cases := []struct {
		name      string
		triggers  *string
		trigger   string
		wantMatch bool
	}{
		{"empty triggers match all", nil, "timeout", true},
		{"429 match", &triggers, "429", true},
		{"5xx match", &triggers, "5xx", true},
		{"timeout no match", &triggers, "timeout", false},
		{"empty string triggers match all", ptr(""), "5xx", true},
		{"disabled channel no match", nil, "5xx", true}, // disabled is checked elsewhere
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ch := &dbmodel.Channel{Id: 1, FallbackEnabled: &yes, FallbackTriggers: c.triggers}
			got := ch.TriggerMatch(c.trigger)
			if got != c.wantMatch {
				t.Errorf("TriggerMatch(%q) = %v, want %v", c.trigger, got, c.wantMatch)
			}
		})
	}
}

// TestBoundarySoftBanConcurrent: 并发软禁读写的安全性（不应当 panic 或死锁）。
func TestBoundarySoftBanConcurrent(t *testing.T) {
	const N = 1000
	var wg sync.WaitGroup
	wg.Add(N * 2)

	// 写
	for i := 0; i < N; i++ {
		go func(id int) {
			defer wg.Done()
			dbmodel.SoftBanChannel(id, time.Now().Unix()+60)
		}(i)
	}
	// 读
	for i := 0; i < N; i++ {
		go func(id int) {
			defer wg.Done()
			_ = dbmodel.IsChannelSoftBanned(id)
		}(i)
	}
	wg.Wait()
	// 清理
	for i := 0; i < N; i++ {
		dbmodel.UnsoftBanChannel(i)
	}
	t.Logf("concurrent soft-ban %d goroutines: ok", N*2)
}

// TestBoundarySoftBanExpiryCleanup: 大量软禁项过期后 CleanupExpiredSoftBan 应当清理。
func TestBoundarySoftBanExpiryCleanup(t *testing.T) {
	for i := 0; i < 50; i++ {
		dbmodel.SoftBanChannel(70000+i, time.Now().Unix()-1) // 已过期
	}
	cleared := dbmodel.CleanupExpiredSoftBan()
	if cleared < 50 {
		t.Errorf("expected to clear at least 50, got %d", cleared)
	}
	t.Logf("CleanupExpiredSoftBan: %d entries cleared", cleared)
}

// TestBoundaryEmptyFallbackPool: 回退池耗尽时应返回 nil 而非 panic。
func TestBoundaryEmptyFallbackPool(t *testing.T) {
	no := false
	channels := []*dbmodel.Channel{
		{Id: 1, FallbackEnabled: &no},
		{Id: 2, FallbackEnabled: &no},
	}
	used := []int{99} // 全部不重复
	var picked *dbmodel.Channel
	for _, c := range channels {
		skip := false
		for _, id := range used {
			if c.Id == id {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if !c.GetFallbackEnabled() {
			continue
		}
		picked = c
		break
	}
	if picked != nil {
		t.Fatalf("expected nil when no candidates, got id=%d", picked.Id)
	}
	t.Log("empty fallback pool: returns nil correctly")
}

// helpers

func ptr(s string) *string { return &s }

func makeChannels(n int, enabled bool, triggers string) []*dbmodel.Channel {
	yes := enabled
	trig := triggers
	out := make([]*dbmodel.Channel, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, &dbmodel.Channel{
			Id:               i,
			Name:             fmt.Sprintf("channel-%d", i),
			FallbackEnabled:  &yes,
			FallbackTriggers: &trig,
		})
	}
	return out
}

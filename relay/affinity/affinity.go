// Package affinity 实现渠道亲和性缓存（sticky channel）。
//
// 核心思路（参考 New API channel_affinity.go）：
// 同一个用户用同一个模型，上次成功走的是 channel X，TTL 内优先复用 X。
// 这避免了无意义跨渠道跳转带来的延迟抖动，也减少了对上游的 cold-start 压力。
//
// 缓存策略：
//   - 纯内存 LRU（sync.Map），不依赖 Redis，不增加网络延迟
//   - 键：tokenId:group:originalModel
//   - 值：channelId + 过期时间
//   - 失败时清除该键，让下次请求重新选
//   - 成功时写入/续期
package affinity

import (
	"sync"
	"time"
)

// entry 是单条亲和性记录。
type entry struct {
	channelId int
	expiresAt int64 // Unix 秒
}

// 全局缓存。
var (
	store sync.Map // key(string) → *entry
)

// Key 生成亲和性缓存的键。
func Key(tokenId int, group string, model string) string {
	// 用 string 做 key 开销可接受，hit 次数远小于请求量
	return itoa(tokenId) + ":" + group + ":" + model
}

// Get 查询亲和性缓存。返回 (channelId, true)，未命中或过期返回 (0, false)。
func Get(key string) (int, bool) {
	v, ok := store.Load(key)
	if !ok {
		return 0, false
	}
	e := v.(*entry)
	if time.Now().Unix() >= e.expiresAt {
		store.Delete(key)
		return 0, false
	}
	return e.channelId, true
}

// Set 写入亲和性缓存。ttlSeconds 为有效期（秒），0 表示使用默认值。
func Set(key string, channelId int, ttlSeconds int) {
	if ttlSeconds <= 0 {
		ttlSeconds = defaultTTL
	}
	store.Store(key, &entry{
		channelId: channelId,
		expiresAt: time.Now().Unix() + int64(ttlSeconds),
	})
}

// Delete 清除指定键的亲和性缓存（请求失败时调用）。
func Delete(key string) {
	store.Delete(key)
}

// DeleteByChannel 清除所有指向该渠道的亲和性缓存（渠道被禁用/删除时调用）。
func DeleteByChannel(channelId int) int {
	cleared := 0
	store.Range(func(k, v any) bool {
		e := v.(*entry)
		if e.channelId == channelId {
			store.Delete(k)
			cleared++
		}
		return true
	})
	return cleared
}

// Cleanup 清理所有过期条目。由后台 goroutine 周期调用。
func Cleanup() int {
	now := time.Now().Unix()
	cleared := 0
	store.Range(func(k, v any) bool {
		if now >= v.(*entry).expiresAt {
			store.Delete(k)
			cleared++
		}
		return true
	})
	return cleared
}

// defaultTTL 默认亲和性有效期（秒）。
const defaultTTL = 300 // 5 分钟

// itoa 简单 int→string，避免 fmt.Sprintf 开销。
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits [12]byte // enough for int
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	pos := len(digits)
	for i > 0 {
		pos--
		digits[pos] = byte(i%10) + '0'
		i /= 10
	}
	if neg {
		pos--
		digits[pos] = '-'
	}
	return string(digits[pos:])
}
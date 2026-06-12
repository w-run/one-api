package model

import (
	"sync"
	"time"
)

// softBannedUntil: channelId → Unix 时间戳（秒），到期后从主选池临时屏蔽。
// 用 sync.Map 而非带 DB 写入，避免给每次限流增加写延迟。
var softBannedUntil sync.Map

// SoftBanChannel: 将指定渠道标记为软禁用至 until（Unix 秒）。
// 重复标记以最新时间为准。
func SoftBanChannel(channelId int, until int64) {
	softBannedUntil.Store(channelId, until)
}

// UnsoftBanChannel: 清除软禁用标记（用于管理员主动恢复或周期清理）。
func UnsoftBanChannel(channelId int) {
	softBannedUntil.Delete(channelId)
}

// IsChannelSoftBanned: 判断渠道当前是否处于软禁用状态。
// 过期项顺手清理，避免 Map 无限增长。
func IsChannelSoftBanned(channelId int) bool {
	v, ok := softBannedUntil.Load(channelId)
	if !ok {
		return false
	}
	until := v.(int64)
	if time.Now().Unix() >= until {
		softBannedUntil.Delete(channelId)
		return false
	}
	return true
}

// CleanupExpiredSoftBan: 由 InitChannelCache 周期调用，清掉过期项。
// 返回清理条数，便于打点。
func CleanupExpiredSoftBan() int {
	now := time.Now().Unix()
	cleared := 0
	softBannedUntil.Range(func(k, v any) bool {
		if now >= v.(int64) {
			softBannedUntil.Delete(k)
			cleared++
		}
		return true
	})
	return cleared
}

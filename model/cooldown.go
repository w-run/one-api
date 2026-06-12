package model

import (
	"fmt"
	"sync"
	"time"
)

// cooldownStore: key → Unix 时间戳（秒）。
// key 格式："channelId"（整渠道）或 "channelId:modelName"（单模型）。
// 单模型冷却：channel X 在 gpt-4o 上收到 429，只冻结 gpt-4o，不影响 claude-3。
// 整渠道冷却：channel X 收到 401/403 等硬错误，整渠道禁用。
var cooldownStore sync.Map

// cooldownKey 生成格式化的 key。
func cooldownKey(channelId int, modelName string) string {
	if modelName == "" {
		return fmt.Sprintf("%d", channelId)
	}
	return fmt.Sprintf("%d:%s", channelId, modelName)
}

// CooldownChannel 整渠道冷却（向后兼容旧 SoftBanChannel）。
func CooldownChannel(channelId int, until int64) {
	cooldownStore.Store(cooldownKey(channelId, ""), until)
}

// CooldownChannelForModel 对指定渠道+模型做冷却。modelName 为空 = 整渠道。
func CooldownChannelForModel(channelId int, modelName string, until int64) {
	cooldownStore.Store(cooldownKey(channelId, modelName), until)
}

// IsChannelCooldown 检查整渠道是否在冷却中。
func IsChannelCooldown(channelId int) bool {
	return isCooldown(cooldownKey(channelId, ""))
}

// IsChannelModelCooldown 检查渠道+模型是否在冷却中。
// 若整渠道被冷却，也返回 true。
func IsChannelModelCooldown(channelId int, modelName string) bool {
	if isCooldown(cooldownKey(channelId, "")) {
		return true
	}
	if modelName == "" {
		return false
	}
	return isCooldown(cooldownKey(channelId, modelName))
}

func isCooldown(key string) bool {
	v, ok := cooldownStore.Load(key)
	if !ok {
		return false
	}
	until := v.(int64)
	if time.Now().Unix() >= until {
		cooldownStore.Delete(key)
		return false
	}
	return true
}

// UncooldownChannel 清除整渠道冷却。
func UncooldownChannel(channelId int) {
	cooldownStore.Delete(cooldownKey(channelId, ""))
}

// UncooldownChannelForModel 清除指定渠道+模型的冷却。
func UncooldownChannelForModel(channelId int, modelName string) {
	cooldownStore.Delete(cooldownKey(channelId, modelName))
}

// CleanupExpiredCooldown 由 InitChannelCache 周期调用，清掉过期项。
func CleanupExpiredCooldown() int {
	now := time.Now().Unix()
	cleared := 0
	cooldownStore.Range(func(k, v any) bool {
		if now >= v.(int64) {
			cooldownStore.Delete(k)
			cleared++
		}
		return true
	})
	return cleared
}

// --- 向后兼容旧 API（标记为 Deprecated） ---

// SoftBanChannel 整渠道软禁用。已废弃，请用 CooldownChannel。
// Deprecated: use CooldownChannel.
func SoftBanChannel(channelId int, until int64) {
	CooldownChannel(channelId, until)
}

// IsChannelSoftBanned 整渠道软禁用检查。已废弃，请用 IsChannelCooldown。
// Deprecated: use IsChannelCooldown.
func IsChannelSoftBanned(channelId int) bool {
	return IsChannelCooldown(channelId)
}

// UnsoftBanChannel 清除整渠道软禁用。已废弃，请用 UncooldownChannel。
// Deprecated: use UncooldownChannel.
func UnsoftBanChannel(channelId int) {
	UncooldownChannel(channelId)
}

// CleanupExpiredSoftBan 清过期软禁用。已废弃，请用 CleanupExpiredCooldown。
// Deprecated: use CleanupExpiredCooldown.
func CleanupExpiredSoftBan() int {
	return CleanupExpiredCooldown()
}
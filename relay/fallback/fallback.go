// Package fallback: 渠道回退决策核心。
// 把错误归类到触发器、判断渠道是否参与回退、执行 429 软禁用。
// 独立成包，避免 common/model 形成循环依赖。
package fallback

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/w-run/mimi-router/common/logger"
	"github.com/w-run/mimi-router/model"
	relaymodel "github.com/w-run/mimi-router/relay/model"
)

// 触发器枚举。与 Channel.FallbackTriggers 字段值对应。
const (
	TriggerRateLimit = "429"     // 上游 429 Too Many Requests
	TriggerServerErr = "5xx"     // 上游 5xx 服务端错误
	TriggerTimeout   = "timeout" // 网络超时 / 连接失败 / context.DeadlineExceeded
)

// ClassifyError: 把任意错误归类到一个触发器。返回 "" 表示不可重试。
func ClassifyError(statusCode int, err error) string {
	// 网络层错误（DoRequest 返回 err != nil）一律视为 timeout
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return TriggerTimeout
		}
		return TriggerTimeout
	}
	switch {
	case statusCode == http.StatusTooManyRequests:
		return TriggerRateLimit
	case statusCode >= 500 && statusCode < 600:
		return TriggerServerErr
	}
	return ""
}

// ShouldFallback: 给定渠道与触发器，判断该渠道是否参与此次回退。
// 规则：
//  1. fallback_enabled=false → 不参与
//  2. trigger 为空 → 不参与（4xx 等不该重试）
//  3. trigger 不在渠道触发器集合中 → 不参与
func ShouldFallback(channel *model.Channel, trigger string) bool {
	if channel == nil {
		return false
	}
	if !channel.GetFallbackEnabled() {
		return false
	}
	if trigger == "" {
		return false
	}
	return channel.TriggerMatch(trigger)
}

// ExtractRetryAfter: 从 *relaymodel.ErrorWithStatusCode 中提取 Retry-After。
// 若 bizErr 不是该类型或非 429，返回 (0, false)。
func ExtractRetryAfter(bizErr *relaymodel.ErrorWithStatusCode) (int, bool) {
	if bizErr == nil {
		return 0, false
	}
	if bizErr.StatusCode != http.StatusTooManyRequests {
		return 0, false
	}
	if bizErr.RetryAfter <= 0 {
		return 0, false
	}
	return bizErr.RetryAfter, true
}

// SoftBanFromError: 收到 429 错误后做软禁用动作。
// 仅在上游明确给出 Retry-After 时启用；未给时不软禁（避免误伤）。
func SoftBanFromError(ctx context.Context, channelId int, bizErr *relaymodel.ErrorWithStatusCode) {
	seconds, ok := ExtractRetryAfter(bizErr)
	if !ok {
		return
	}
	until := time.Now().Unix() + int64(seconds)
	model.SoftBanChannel(channelId, until)
	logger.Debugf(ctx, "channel #%d soft-banned for %ds (until %d)", channelId, seconds, until)
}

// ParseRetryAfter: 解析 HTTP Retry-After 头的整数值。
// 支持整数秒和 HTTP-date；解析失败返回 0。
func ParseRetryAfter(value string) int {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0
		}
		return seconds
	}
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		diff := int(time.Until(t).Seconds())
		if diff < 0 {
			return 0
		}
		return diff
	}
	return 0
}

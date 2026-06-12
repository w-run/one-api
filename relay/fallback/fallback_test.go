package fallback

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/w-run/mimi-router/model"
	relaymodel "github.com/w-run/mimi-router/relay/model"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		err        error
		want       string
	}{
		{"429 too many requests", http.StatusTooManyRequests, nil, TriggerRateLimit},
		{"502 bad gateway", http.StatusBadGateway, nil, TriggerServerErr},
		{"503 service unavailable", http.StatusServiceUnavailable, nil, TriggerServerErr},
		{"504 gateway timeout", http.StatusGatewayTimeout, nil, TriggerServerErr},
		{"400 bad request", http.StatusBadRequest, nil, ""},
		{"401 unauthorized", http.StatusUnauthorized, nil, ""},
		{"200 ok", http.StatusOK, nil, ""},
		{"network deadline", 0, context.DeadlineExceeded, TriggerTimeout},
		{"network canceled", 0, context.Canceled, TriggerTimeout},
		{"network other", 0, errors.New("dial tcp: connection refused"), TriggerTimeout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ClassifyError(c.statusCode, c.err)
			if got != c.want {
				t.Errorf("ClassifyError(%d, %v) = %q, want %q", c.statusCode, c.err, got, c.want)
			}
		})
	}
}

func TestShouldFallback(t *testing.T) {
	yes := true
	no := false
	triggers := "429,5xx"

	cases := []struct {
		name    string
		channel *model.Channel
		trigger string
		want    bool
	}{
		{"nil channel", nil, TriggerRateLimit, false},
		{"fallback disabled", &model.Channel{FallbackEnabled: &no}, TriggerRateLimit, false},
		{"empty trigger", &model.Channel{FallbackEnabled: &yes}, "", false},
		{"empty triggers match all 429", &model.Channel{FallbackEnabled: &yes, FallbackTriggers: nil}, TriggerRateLimit, true},
		{"empty triggers match all 5xx", &model.Channel{FallbackEnabled: &yes, FallbackTriggers: nil}, TriggerServerErr, true},
		{"matching trigger 429", &model.Channel{FallbackEnabled: &yes, FallbackTriggers: &triggers}, TriggerRateLimit, true},
		{"matching trigger 5xx", &model.Channel{FallbackEnabled: &yes, FallbackTriggers: &triggers}, TriggerServerErr, true},
		{"non-matching trigger timeout", &model.Channel{FallbackEnabled: &yes, FallbackTriggers: &triggers}, TriggerTimeout, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ShouldFallback(c.channel, c.trigger)
			if got != c.want {
				t.Errorf("ShouldFallback() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestExtractRetryAfter(t *testing.T) {
	cases := []struct {
		name    string
		bizErr  *relaymodel.ErrorWithStatusCode
		wantVal int
		wantOk  bool
	}{
		{"nil", nil, 0, false},
		{"non-429", &relaymodel.ErrorWithStatusCode{StatusCode: 500, RetryAfter: 30}, 0, false},
		{"429 without retry-after", &relaymodel.ErrorWithStatusCode{StatusCode: 429, RetryAfter: 0}, 0, false},
		{"429 with retry-after", &relaymodel.ErrorWithStatusCode{StatusCode: 429, RetryAfter: 60}, 60, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotVal, gotOk := ExtractRetryAfter(c.bizErr)
			if gotVal != c.wantVal || gotOk != c.wantOk {
				t.Errorf("ExtractRetryAfter() = (%d, %v), want (%d, %v)", gotVal, gotOk, c.wantVal, c.wantOk)
			}
		})
	}
}

func TestSoftBanRoundTrip(t *testing.T) {
	channelId := 99999
	// ensure clean
	model.UnsoftBanChannel(channelId)

	// 软禁用 2 秒
	until := time.Now().Unix() + 2
	model.SoftBanChannel(channelId, until)

	if !model.IsChannelSoftBanned(channelId) {
		t.Error("expected channel to be soft-banned")
	}

	// 等待过期
	time.Sleep(2100 * time.Millisecond)

	if model.IsChannelSoftBanned(channelId) {
		t.Error("expected channel to be un-banned after expiration")
	}
}

func TestCleanupExpiredSoftBan(t *testing.T) {
	// set 3 entries, all expired
	for i := 0; i < 3; i++ {
		model.SoftBanChannel(80000+i, time.Now().Unix()-1)
	}
	cleared := model.CleanupExpiredSoftBan()
	if cleared < 3 {
		t.Errorf("expected to clear at least 3 entries, got %d", cleared)
	}
}

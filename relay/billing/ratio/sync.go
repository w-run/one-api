package ratio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/w-run/one-api/common/logger"
)

const (
	// 1 ratio unit = $0.002 / 1K tokens
	ratioUnitPrice = 0.002
)

// openRouterModelPricing matches OpenRouter API pricing field
type openRouterModelPricing struct {
	Prompt          string `json:"prompt"`
	Completion      string `json:"completion"`
	InputCacheRead  string `json:"input_cache_read,omitempty"`
	InputCacheWrite string `json:"input_cache_write,omitempty"`
}

type openRouterModelItem struct {
	ID      string                  `json:"id"`
	Pricing openRouterModelPricing  `json:"pricing"`
}

type openRouterResponse struct {
	Data []openRouterModelItem `json:"data"`
}

// SyncModelRatiosFromURL fetches pricing from OpenRouter-style API
// and merges into ModelRatio map
func SyncModelRatiosFromURL(url string) (int, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API 返回错误 (%d): %s", resp.StatusCode, string(body))
	}

	// Try OpenRouter format first
	var orResp openRouterResponse
	if err := json.Unmarshal(body, &orResp); err == nil && len(orResp.Data) > 0 && orResp.Data[0].Pricing.Prompt != "" {
		return syncFromOpenRouter(orResp.Data)
	}

	// Try standard format
	var stdResp struct {
		Models []struct {
			ID         string  `json:"id"`
			InputPrice float64 `json:"input_price"`
			OutputPrice float64 `json:"output_price"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &stdResp); err == nil && len(stdResp.Models) > 0 {
		return syncFromStandardFormat(stdResp.Models)
	}

	return 0, fmt.Errorf("无法识别响应格式")
}

func syncFromOpenRouter(models []openRouterModelItem) (int, error) {
	modelRatioLock.Lock()
	defer modelRatioLock.Unlock()

	count := 0
	for _, m := range models {
		modelID := m.ID
		promptPrice := parsePrice(m.Pricing.Prompt)
		completionPrice := parsePrice(m.Pricing.Completion)

		if promptPrice <= 0 {
			continue
		}

		// Use average of prompt and completion price for simplicity
		avgPrice := promptPrice
		if completionPrice > 0 {
			avgPrice = (promptPrice + completionPrice) / 2
		}

		// Convert per-token USD to One API ratio
		// 1 ratio = $0.002 / 1K tokens
		// price per 1K tokens = avgPrice * 1000
		// ratio = (avgPrice * 1000) / 0.002
		ratio := (avgPrice * 1000) / ratioUnitPrice

		// Round to reasonable precision
		ratio = float64(int(ratio*10000)) / 10000

		ModelRatio[modelID] = ratio
		count++
	}

	logger.SysLogf("从 OpenRouter 同步了 %d 个模型的倍率", count)
	return count, nil
}

func syncFromStandardFormat(models []struct {
	ID          string  `json:"id"`
	InputPrice  float64 `json:"input_price"`
	OutputPrice float64 `json:"output_price"`
}) (int, error) {
	modelRatioLock.Lock()
	defer modelRatioLock.Unlock()

	count := 0
	for _, m := range models {
		if m.InputPrice <= 0 {
			continue
		}
		avgPrice := m.InputPrice
		if m.OutputPrice > 0 {
			avgPrice = (m.InputPrice + m.OutputPrice) / 2
		}
		ratio := (avgPrice * 1000) / ratioUnitPrice
		ratio = float64(int(ratio*10000)) / 10000
		ModelRatio[m.ID] = ratio
		count++
	}

	logger.SysLogf("从自定义源同步了 %d 个模型的倍率", count)
	return count, nil
}

func parsePrice(s string) float64 {
	if s == "" || s == "0" {
		return 0
	}
	var price float64
	if _, err := fmt.Sscanf(s, "%f", &price); err != nil {
		return 0
	}
	return price
}

// SyncFromOpenRouter is a convenience function that uses the default OpenRouter URL
func SyncFromOpenRouter() (int, error) {
	return SyncModelRatiosFromURL("https://openrouter.ai/api/v1/models")
}
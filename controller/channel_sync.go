package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w-run/one-api/model"
	"github.com/w-run/one-api/relay/channeltype"
)

// openRouterModelResponse matches OpenRouter /api/v1/models response
type openRouterModelResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Known OpenAI-compatible provider base URLs
// Extracted from OpenRouter model IDs (format: "provider/model-name")
var providerBaseURLs = map[string]string{
	// 主流
	"openai":        "https://api.openai.com",
	"anthropic":     "https://api.anthropic.com",
	"google":        "https://generativelanguage.googleapis.com/v1beta/openai/",
	"mistral":       "https://api.mistral.ai",
	"cohere":        "https://api.cohere.ai",
	"deepseek":      "https://api.deepseek.com",
	"xai":           "https://api.x.ai",
	"meta-llama":    "", // 需用户自行配置
	"microsoft":     "https://models.inference.ai.azure.com",
	"qwen":          "https://dashscope.aliyuncs.com/compatible-mode/v1",

	// 新增
	"nvidia":        "https://integrate.api.nvidia.com/v1",
	"perplexity":    "https://api.perplexity.ai",
	"cerebras":      "https://api.cerebras.ai/v1",
	"github":        "https://models.inference.ai.azure.com",

	// 社区热门
	"groq":          "https://api.groq.com/openai",
	"together":      "https://api.together.xyz",
	"fireworks":     "https://api.fireworks.ai/inference",
	"liquid":        "https://api.liquid.ai/v1",
	"inflection":    "https://api.inflection.ai/v1",
	"reka":          "https://api.reka.ai/v1",
	"writer":        "https://api.writer.com/v1",
	"ai21":          "https://api.ai21.com/studio/v1",

	// 国内
	"01-ai":         "", // Yi 系列 → 通过 SiliconFlow 或自定义
	"deepseek-ai":   "https://api.deepseek.com",
	"siliconflow":   "https://api.siliconflow.cn",
	"minimax":       "https://api.minimax.chat",
	"lingyiwanwu":   "https://api.lingyiwanwu.com",
	"stepfun":       "https://api.stepfun.com",
	"zhipu":         "https://open.bigmodel.cn",
	"baidu":         "https://qianfan.baidubce.com",
	"ali":           "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"tencent":       "https://hunyuan.tencentcloudapi.com",
}

// providerDisplayNames maps provider keys to human-readable names
var providerDisplayNames = map[string]string{
	"openai":      "OpenAI",
	"anthropic":   "Anthropic",
	"google":      "Google Gemini",
	"mistral":     "Mistral AI",
	"cohere":      "Cohere",
	"deepseek":    "DeepSeek",
	"xai":         "xAI (Grok)",
	"microsoft":   "Microsoft (GitHub Models)",
	"qwen":        "阿里通义千问",
	"nvidia":      "NVIDIA NIM",
	"perplexity":  "Perplexity",
	"cerebras":    "Cerebras",
	"github":      "GitHub Models",
	"groq":        "Groq",
	"together":    "Together AI",
	"fireworks":   "Fireworks AI",
	"liquid":      "Liquid AI",
	"inflection":  "Inflection AI",
	"reka":        "Reka AI",
	"writer":      "Writer",
	"ai21":        "AI21 Labs",
	"siliconflow": "SiliconFlow",
	"minimax":     "MiniMax",
	"lingyiwanwu": "零一万物",
	"stepfun":     "阶跃星辰",
	"zhipu":       "智谱 AI",
	"baidu":       "百度千帆",
	"ali":         "阿里通义千问",
	"tencent":     "腾讯混元",
}

func SyncProviders(c *gin.Context) {
	// 1. Fetch model list from OpenRouter
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建请求失败：" + err.Error(),
		})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请求 OpenRouter API 失败：" + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "读取响应失败：" + err.Error(),
		})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("OpenRouter API 返回错误（%d）：%s", resp.StatusCode, string(body)),
		})
		return
	}

	var modelList openRouterModelResponse
	if err := json.Unmarshal(body, &modelList); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "解析响应失败：" + err.Error(),
		})
		return
	}

	// 2. Extract unique provider names from model IDs (format: "provider/model-name")
	providerSet := make(map[string]bool)
	for _, m := range modelList.Data {
		parts := strings.SplitN(m.ID, "/", 2)
		if len(parts) == 2 && parts[0] != "" {
			providerSet[parts[0]] = true
		}
	}

	// 3. Load existing channels for dedup
	existing, err := model.GetAllChannels(0, 10000, "all")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "读取现有渠道失败：" + err.Error(),
		})
		return
	}
	existingNames := make(map[string]bool)
	for _, ch := range existing {
		existingNames[ch.Name] = true
	}

	// 4. Create channels for new providers with known base URLs
	var created []string
	var skipped []string
	var noEndpoint []string

	for provider := range providerSet {
		baseURL, known := providerBaseURLs[provider]
		if !known || baseURL == "" {
			noEndpoint = append(noEndpoint, provider)
			continue
		}

		displayName := providerDisplayNames[provider]
		if displayName == "" {
			displayName = provider
		}
		channelName := fmt.Sprintf("[自动同步] %s", displayName)

		if existingNames[channelName] {
			skipped = append(skipped, provider)
			continue
		}

		channel := model.Channel{
			Type:        channeltype.OpenAICompatible,
			Name:        channelName,
			BaseURL:     &baseURL,
			Status:      model.ChannelStatusEnabled,
		}
		if err := channel.Insert(); err != nil {
			skipped = append(skipped, provider+"("+err.Error()+")")
			continue
		}
		created = append(created, provider)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"message":    fmt.Sprintf("同步完成：新增 %d，跳过 %d（已存在），%d 无已知 API 地址", len(created), len(skipped), len(noEndpoint)),
		"created":    created,
		"skipped":    skipped,
		"no_endpoint": noEndpoint,
	})
}
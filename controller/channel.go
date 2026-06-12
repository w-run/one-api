package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w-run/mimi-router/common/config"
	"github.com/w-run/mimi-router/common/helper"
	"github.com/w-run/mimi-router/model"
	"github.com/w-run/mimi-router/relay/channeltype"
)

func GetAllChannels(c *gin.Context) {
	p, _ := strconv.Atoi(c.Query("p"))
	if p < 0 {
		p = 0
	}
	channels, err := model.GetAllChannels(p*config.ItemsPerPage, config.ItemsPerPage, "limited")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
	return
}

func SearchChannels(c *gin.Context) {
	keyword := c.Query("keyword")
	channels, err := model.SearchChannels(keyword)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channels,
	})
	return
}

func GetChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel, err := model.GetChannelById(id, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
	return
}

func AddChannel(c *gin.Context) {
	channel := model.Channel{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channel.CreatedTime = helper.GetTimestamp()
	keys := strings.Split(channel.Key, "\n")
	channels := make([]model.Channel, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		localChannel := channel
		localChannel.Key = key
		channels = append(channels, localChannel)
	}
	err = model.BatchInsertChannels(channels)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteChannel(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	channel := model.Channel{Id: id}
	err := channel.Delete()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
	return
}

func DeleteDisabledChannel(c *gin.Context) {
	rows, err := model.DeleteDisabledChannel()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    rows,
	})
	return
}

func UpdateChannel(c *gin.Context) {
	channel := model.Channel{}
	err := c.ShouldBindJSON(&channel)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = channel.Update()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    channel,
	})
	return
}

type fetchModelsRequest struct {
	Type      int    `json:"type"`
	BaseURL   string `json:"base_url"`
	Key       string `json:"key"`
	ChannelId int    `json:"channel_id"`
}

type openAIModelListItem struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type openAIModelListResponse struct {
	Object string                `json:"object"`
	Data   []openAIModelListItem `json:"data"`
}

func FetchChannelModels(c *gin.Context) {
	var req fetchModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// If key is not provided but channel_id is, fetch key from database
	if req.Key == "" && req.ChannelId != 0 {
		log.Printf("[FetchChannelModels] 前端未提供密钥，尝试从数据库获取，channel_id=%d", req.ChannelId)
		channel, err := model.GetChannelById(req.ChannelId, true)
		if err != nil {
			log.Printf("[FetchChannelModels] 从数据库获取渠道失败，channel_id=%d, 错误: %v", req.ChannelId, err)
		} else if channel.Key == "" {
			log.Printf("[FetchChannelModels] 渠道存在但密钥为空，channel_id=%d, channel_name=%s", req.ChannelId, channel.Name)
		} else {
			req.Key = channel.Key
			log.Printf("[FetchChannelModels] 成功从数据库获取密钥，channel_id=%d, channel_name=%s, key_length=%d", req.ChannelId, channel.Name, len(req.Key))
		}
	} else if req.Key == "" && req.ChannelId == 0 {
		log.Printf("[FetchChannelModels] 前端未提供密钥且未提供 channel_id，无法回退获取密钥")
	}

	// Determine base URL
	baseURL := strings.TrimRight(req.BaseURL, "/")
	if baseURL == "" {
		if req.Type >= 0 && req.Type < len(channeltype.ChannelBaseURLs) {
			baseURL = channeltype.ChannelBaseURLs[req.Type]
		}
	}
	if baseURL == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无法确定渠道 API 地址，请先填写渠道 API 地址",
		})
		return
	}

	// Build model list URL (OpenAI-compatible)
	apiURL := fmt.Sprintf("%s/v1/models", baseURL)
	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建请求失败：" + err.Error(),
		})
		return
	}
	if req.Key != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Key)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请求失败：" + err.Error(),
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
			"message": fmt.Sprintf("API 返回错误（%d）：%s", resp.StatusCode, string(body)),
		})
		return
	}

	var modelList openAIModelListResponse
	if err := json.Unmarshal(body, &modelList); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "解析响应失败：" + err.Error(),
		})
		return
	}

	models := make([]string, 0, len(modelList.Data))
	for _, m := range modelList.Data {
		models = append(models, m.Id)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    models,
	})
}

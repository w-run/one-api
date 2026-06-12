package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/w-run/mimi-router/common/ctxkey"
	"github.com/w-run/mimi-router/common/logger"
	"github.com/w-run/mimi-router/model"
	"github.com/w-run/mimi-router/relay/affinity"
	"github.com/w-run/mimi-router/relay/channeltype"
)

type ModelRequest struct {
	Model string `json:"model" form:"model"`
}

func Distribute() func(c *gin.Context) {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		userId := c.GetInt(ctxkey.Id)
		userGroup, backupGroup, _ := model.CacheGetUserGroupWithBackup(userId)
		c.Set(ctxkey.Group, userGroup)
		c.Set(ctxkey.BackupGroup, backupGroup)
		var requestModel string
		var channel *model.Channel
		channelId, ok := c.Get(ctxkey.SpecificChannelId)
		if ok {
			id, err := strconv.Atoi(channelId.(string))
			if err != nil {
				abortWithMessage(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			channel, err = model.GetChannelById(id, true)
			if err != nil {
				abortWithMessage(c, http.StatusBadRequest, "无效的渠道 Id")
				return
			}
			if channel.Status != model.ChannelStatusEnabled {
				abortWithMessage(c, http.StatusForbidden, "该渠道已被禁用")
				return
			}
		} else {
			requestModel = c.GetString(ctxkey.RequestModel)
			var err error
			// 渠道亲和性：先查上次该用户+模型成功的渠道
			tokenId := c.GetInt(ctxkey.TokenId)
			affinityKey := affinity.Key(tokenId, userGroup, requestModel)
			if hitId, ok := affinity.Get(affinityKey); ok {
				affCh, e := model.GetChannelById(hitId, true)
				if e == nil && affCh.Status == model.ChannelStatusEnabled && !model.IsChannelModelCooldown(hitId, requestModel) {
					logger.Debugf(ctx, "affinity hit: channel #%d for model %s", hitId, requestModel)
					channel = affCh
				} else {
					// 亲和渠道不可用，清除
					affinity.Delete(affinityKey)
				}
			}
			if channel == nil {
				channel, err = model.CacheGetRandomSatisfiedChannel(userGroup, requestModel, false)
			}
			if err != nil {
				message := fmt.Sprintf("当前分组 %s 下对于模型 %s 无可用渠道", userGroup, requestModel)
				if channel != nil {
					logger.SysError(fmt.Sprintf("渠道不存在：%d", channel.Id))
					message = "数据库一致性已被破坏，请联系管理员"
				}
				abortWithMessage(c, http.StatusServiceUnavailable, message)
				return
			}
		}
		logger.Debugf(ctx, "user id %d, user group: %s, request model: %s, using channel #%d", userId, userGroup, requestModel, channel.Id)
		SetupContextForSelectedChannel(c, channel, requestModel)
		c.Next()
	}
}

func SetupContextForSelectedChannel(c *gin.Context, channel *model.Channel, modelName string) {
	c.Set(ctxkey.Channel, channel.Type)
	c.Set(ctxkey.ChannelId, channel.Id)
	c.Set(ctxkey.ChannelName, channel.Name)
	if channel.SystemPrompt != nil && *channel.SystemPrompt != "" {
		c.Set(ctxkey.SystemPrompt, *channel.SystemPrompt)
	}
	c.Set(ctxkey.ModelMapping, channel.GetModelMapping())
	c.Set(ctxkey.OriginalModel, modelName) // for retry
	c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", channel.Key))
	c.Set(ctxkey.BaseURL, channel.GetBaseURL())
	cfg, _ := channel.LoadConfig()
	// this is for backward compatibility
	if channel.Other != nil {
		switch channel.Type {
		case channeltype.Azure:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *channel.Other
			}
		case channeltype.Xunfei:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *channel.Other
			}
		case channeltype.Gemini:
			if cfg.APIVersion == "" {
				cfg.APIVersion = *channel.Other
			}
		case channeltype.AIProxyLibrary:
			if cfg.LibraryID == "" {
				cfg.LibraryID = *channel.Other
			}
		case channeltype.Ali:
			if cfg.Plugin == "" {
				cfg.Plugin = *channel.Other
			}
		}
	}
	c.Set(ctxkey.Config, cfg)
}

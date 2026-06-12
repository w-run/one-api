package common

import (
	"fmt"

	"github.com/w-run/mimi-router/common/config"
)

func LogQuota(quota int64) string {
	if config.DisplayInCurrencyEnabled {
		// 默认以人民币（CNY）展示；若汇率变更请同步 QuotaPerUnit 注释。
		return fmt.Sprintf("￥%.6f 额度", float64(quota)/config.QuotaPerUnit)
	}
	return fmt.Sprintf("%d 点额度", quota)
}

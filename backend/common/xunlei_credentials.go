package pan

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseXunleiRefreshToken accepts the preferred raw refresh_token and the
// legacy JSON form used by earlier releases.
func ParseXunleiRefreshToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("refresh_token 为空")
	}
	if !strings.HasPrefix(value, "{") {
		return value, nil
	}

	var payload struct {
		RefreshToken string `json:"refresh_token"`
		Token        *struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"token"`
	}
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return "", fmt.Errorf("refresh_token JSON 格式错误: %w", err)
	}
	if payload.RefreshToken != "" {
		return strings.TrimSpace(payload.RefreshToken), nil
	}
	if payload.Token != nil && payload.Token.RefreshToken != "" {
		return strings.TrimSpace(payload.Token.RefreshToken), nil
	}
	return "", fmt.Errorf("JSON 中缺少 refresh_token")
}

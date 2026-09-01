package pan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	xunleiWebClientID          = "Xqp0kJBXWhwaTpB6"
	xunleiWebDeviceID          = "925b7631473a13716b791d7f28289cad"
	xunleiWebClientVersion     = "1.45.0"
	xunleiWebPackageName       = "pan.xunlei.com"
	xunleiWebCaptchaSign       = "1.fe2108ad808a74c9ac0243309242726c"
	xunleiWebCaptchaTimestamp  = "1645241033384"
	xunleiWebUserAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"
	xunleiCaptchaUserAgent     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	xunleiAuthTokenURL         = "https://xluser-ssl.xunlei.com/v1/auth/token"
	xunleiCaptchaInitURL       = "https://xluser-ssl.xunlei.com/v1/shield/captcha/init"
	xunleiDefaultCaptchaAction = "get:/drive/v1/share"
)

// captchaInit initializes the web captcha token used by drive API requests.
// The initial payload matches pan.xunlei.com; retries use the rejected API's
// method/path and the authenticated user ID, as required by Xunlei error 9.
func (x *XunleiPanService) captchaInit(action, userID string) (string, error) {
	if action == "" {
		action = xunleiDefaultCaptchaAction
	}
	if userID == "" {
		userID = "0"
	}
	body := map[string]interface{}{
		"client_id": xunleiWebClientID,
		"action":    action,
		"device_id": xunleiWebDeviceID,
		"meta": map[string]interface{}{
			"username":       "",
			"phone_number":   "",
			"email":          "",
			"package_name":   xunleiWebPackageName,
			"client_version": xunleiWebClientVersion,
			"captcha_sign":   xunleiWebCaptchaSign,
			"timestamp":      xunleiWebCaptchaTimestamp,
			"user_id":        userID,
		},
	}
	if x.extra.Captcha != nil && x.extra.Captcha.CaptchaToken != "" {
		body["captcha_token"] = x.extra.Captcha.CaptchaToken
	}

	resp, err := x.sendXunleiAuthRequest(xunleiCaptchaInitURL, body, map[string]string{
		"User-Agent": xunleiCaptchaUserAgent,
	})
	if err != nil {
		return "", fmt.Errorf("captcha init 请求失败: %w", err)
	}
	if verifyURL, _ := resp["url"].(string); verifyURL != "" {
		return "", fmt.Errorf("迅雷账号触发安全验证，请先在迅雷网页版完成验证: %s", verifyURL)
	}

	token, _ := resp["captcha_token"].(string)
	if token == "" {
		return "", fmt.Errorf("captcha init 未返回 captcha_token")
	}
	expiresIn := int64(3600)
	if exp, ok := resp["expires_in"].(float64); ok {
		expiresIn = int64(exp)
	}
	if x.extra.Captcha == nil {
		x.extra.Captcha = &CaptchaData{}
	}
	x.extra.Captcha.CaptchaToken = token
	x.extra.Captcha.ExpiresAt = time.Now().Unix() + expiresIn - 10
	x.persistExtra()
	return token, nil
}

// LoginByRefreshToken exchanges a pan.xunlei.com refresh_token for an access token.
func (x *XunleiPanService) LoginByRefreshToken(refreshToken string) (XunleiTokenData, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return XunleiTokenData{}, fmt.Errorf("refresh_token 为空")
	}

	token, err := x.GetAccessTokenByRefreshToken(refreshToken)
	if err != nil {
		return XunleiTokenData{}, err
	}
	x.extra.AuthClientID = xunleiWebClientID
	x.extra.Token = &token
	x.extra.Captcha = nil
	return token, nil
}

// sendXunleiAuthRequest sends a JSON request with the same identity headers as
// pan.xunlei.com. It deliberately avoids logging token-bearing response bodies.
func (x *XunleiPanService) sendXunleiAuthRequest(reqURL string, data map[string]interface{}, extraHeaders map[string]string) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, reqURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://pan.xunlei.com")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", "https://pan.xunlei.com/")
	req.Header.Set("User-Agent", xunleiWebUserAgent)
	req.Header.Set("X-Client-Id", xunleiWebClientID)
	req.Header.Set("X-Device-Id", xunleiWebDeviceID)
	for key, value := range extraHeaders {
		req.Header.Set(key, value)
	}

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("迅雷认证请求失败: HTTP %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("迅雷认证响应解析失败: %w", err)
	}
	return result, nil
}

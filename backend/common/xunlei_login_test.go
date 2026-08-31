package pan

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(body string) *http.Response {
	return jsonResponseWithStatus(http.StatusOK, body)
}

func jsonResponseWithStatus(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestLoginByRefreshTokenUsesWebIdentity(t *testing.T) {
	svc := NewXunleiPanService(nil)
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != xunleiAuthTokenURL {
			t.Fatalf("请求地址 = %s, want %s", req.URL, xunleiAuthTokenURL)
		}
		if req.Header.Get("X-Client-Id") != xunleiWebClientID {
			t.Fatalf("X-Client-Id = %q", req.Header.Get("X-Client-Id"))
		}
		if req.Header.Get("X-Device-Id") != xunleiWebDeviceID {
			t.Fatalf("X-Device-Id = %q", req.Header.Get("X-Device-Id"))
		}
		if req.Header.Get("Origin") != "https://pan.xunlei.com" {
			t.Fatalf("Origin = %q", req.Header.Get("Origin"))
		}

		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("解析请求体失败: %v", err)
		}
		if body["client_id"] != xunleiWebClientID {
			t.Fatalf("client_id = %v", body["client_id"])
		}
		if body["grant_type"] != "refresh_token" || body["refresh_token"] != "old-refresh" {
			t.Fatalf("认证请求体不正确: %#v", body)
		}
		if _, exists := body["client_secret"]; exists {
			t.Fatal("网页端 token 刷新不应携带 client_secret")
		}
		return jsonResponse(`{"access_token":"access","refresh_token":"new-refresh","expires_in":7200,"sub":"user-1"}`), nil
	})}

	token, err := svc.LoginByRefreshToken(" old-refresh ")
	if err != nil {
		t.Fatalf("LoginByRefreshToken error: %v", err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "new-refresh" {
		t.Fatalf("token = %#v", token)
	}
	if svc.extra.AuthClientID != xunleiWebClientID || svc.extra.Token == nil {
		t.Fatalf("认证状态未缓存: %#v", svc.extra)
	}
}

func TestCaptchaInitUsesReferencePayload(t *testing.T) {
	svc := NewXunleiPanService(nil)
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != xunleiCaptchaInitURL {
			t.Fatalf("请求地址 = %s, want %s", req.URL, xunleiCaptchaInitURL)
		}
		if req.Header.Get("User-Agent") != xunleiCaptchaUserAgent {
			t.Fatalf("captcha User-Agent = %q", req.Header.Get("User-Agent"))
		}

		var body struct {
			ClientID string                 `json:"client_id"`
			Action   string                 `json:"action"`
			DeviceID string                 `json:"device_id"`
			Meta     map[string]interface{} `json:"meta"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatalf("解析请求体失败: %v", err)
		}
		if body.ClientID != xunleiWebClientID || body.DeviceID != xunleiWebDeviceID {
			t.Fatalf("captcha 身份不正确: %#v", body)
		}
		if body.Action != xunleiDefaultCaptchaAction {
			t.Fatalf("action = %q", body.Action)
		}
		if body.Meta["package_name"] != xunleiWebPackageName || body.Meta["client_version"] != xunleiWebClientVersion {
			t.Fatalf("captcha meta 客户端信息不正确: %#v", body.Meta)
		}
		if body.Meta["captcha_sign"] != xunleiWebCaptchaSign || body.Meta["timestamp"] != xunleiWebCaptchaTimestamp {
			t.Fatalf("captcha meta 签名不正确: %#v", body.Meta)
		}
		if body.Meta["user_id"] != "0" {
			t.Fatalf("user_id = %#v", body.Meta["user_id"])
		}
		return jsonResponse(`{"captcha_token":"captcha","expires_in":300}`), nil
	})}

	token, err := svc.getCaptchaToken()
	if err != nil {
		t.Fatalf("getCaptchaToken error: %v", err)
	}
	if token != "captcha" || svc.extra.Captcha == nil || svc.extra.Captcha.CaptchaToken != "captcha" {
		t.Fatalf("captcha 状态未缓存: %#v", svc.extra.Captcha)
	}
}

func TestRequestXunleiApiRefreshesCaptchaForRejectedAction(t *testing.T) {
	svc := NewXunleiPanService(nil)
	svc.extra.Token = &XunleiTokenData{UserId: "user-1"}
	svc.extra.Captcha = &CaptchaData{CaptchaToken: "stale-captcha"}
	apiAttempts := 0
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.String() {
		case "https://api-pan.xunlei.com/drive/v1/files":
			apiAttempts++
			if apiAttempts == 1 {
				if req.Header.Get("x-captcha-token") != "stale-captcha" {
					t.Fatalf("首次 captcha token = %q", req.Header.Get("x-captcha-token"))
				}
				return jsonResponseWithStatus(http.StatusBadRequest, `{"error":"captcha_invalid","error_code":9}`), nil
			}
			if req.Header.Get("x-captcha-token") != "fresh-captcha" {
				t.Fatalf("重试 captcha token = %q", req.Header.Get("x-captcha-token"))
			}
			return jsonResponse(`{"id":"folder-1"}`), nil
		case xunleiCaptchaInitURL:
			var body struct {
				Action       string                 `json:"action"`
				CaptchaToken string                 `json:"captcha_token"`
				Meta         map[string]interface{} `json:"meta"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatalf("解析 captcha 请求失败: %v", err)
			}
			if body.Action != "POST:/drive/v1/files" {
				t.Fatalf("action = %q", body.Action)
			}
			if body.CaptchaToken != "stale-captcha" || body.Meta["user_id"] != "user-1" {
				t.Fatalf("captcha 刷新上下文不正确: %#v", body)
			}
			return jsonResponse(`{"captcha_token":"fresh-captcha","expires_in":300}`), nil
		default:
			t.Fatalf("未预期请求: %s", req.URL)
			return nil, nil
		}
	})}

	result, err := svc.requestXunleiApi(
		"https://api-pan.xunlei.com/drive/v1/files",
		http.MethodPost,
		map[string]interface{}{"kind": "drive#folder"},
		nil,
		map[string]string{"x-captcha-token": "stale-captcha"},
	)
	if err != nil {
		t.Fatalf("requestXunleiApi error: %v", err)
	}
	if result["id"] != "folder-1" || apiAttempts != 2 {
		t.Fatalf("result=%#v attempts=%d", result, apiAttempts)
	}
}

func TestParseXunleiRefreshToken(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "raw", value: " a1.raw-token ", want: "a1.raw-token"},
		{name: "legacy json", value: `{"refresh_token":"a1.legacy","client_type":"android"}`, want: "a1.legacy"},
		{name: "extra json", value: `{"token":{"refresh_token":"a1.extra"}}`, want: "a1.extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseXunleiRefreshToken(tt.value)
			if err != nil {
				t.Fatalf("ParseXunleiRefreshToken error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

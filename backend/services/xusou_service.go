package services

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultXusouSearchEndpoint = "https://xusou.cn/api/other/web_search"
	defaultXusouResolvePath    = "/api/other/save_url"
	maxXusouEventSize          = 1 << 20
)

// XusouSearchResult is the small result envelope emitted by xusou's SSE API.
// URL is intentionally kept encoded until a visitor asks to get the link.
type XusouSearchResult struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	IsType     int    `json:"is_type"`
	LineName   string `json:"line_name,omitempty"`
	LineWeight int    `json:"line_weight,omitempty"`
	LineIndex  int    `json:"line_index,omitempty"`
}

type xusouResolveResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"data"`
}

// XusouClient wraps xusou's search and temporary-link endpoints. Keeping this
// server-side prevents the public application from becoming a general proxy.
type XusouClient struct {
	searchEndpoint string
	resolveURL     string
	httpClient     *http.Client
}

func NewXusouClient() *XusouClient {
	return NewXusouClientWithEndpoints(defaultXusouSearchEndpoint, "", nil)
}

// NewXusouClientWithEndpoint is useful for isolated integration tests.
func NewXusouClientWithEndpoint(endpoint string, client *http.Client) *XusouClient {
	return NewXusouClientWithEndpoints(endpoint, "", client)
}

// NewXusouClientWithEndpoints allows tests to provide separate search and
// resolve handlers while retaining the production defaults.
func NewXusouClientWithEndpoints(searchEndpoint, resolveEndpoint string, client *http.Client) *XusouClient {
	if client == nil {
		client = &http.Client{Timeout: 35 * time.Second}
	}
	if strings.TrimSpace(resolveEndpoint) == "" {
		resolveEndpoint = siblingEndpoint(searchEndpoint, defaultXusouResolvePath)
	}
	return &XusouClient{
		searchEndpoint: strings.TrimSpace(searchEndpoint),
		resolveURL:     strings.TrimSpace(resolveEndpoint),
		httpClient:     client,
	}
}

func siblingEndpoint(rawEndpoint, path string) string {
	parsed, err := url.Parse(rawEndpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

// Search reads one xusou SSE stream to completion.
func (x *XusouClient) Search(ctx context.Context, query string, isType int) ([]XusouSearchResult, error) {
	if strings.TrimSpace(x.searchEndpoint) == "" {
		return nil, fmt.Errorf("xusou 搜索地址未配置")
	}
	parsed, err := url.Parse(x.searchEndpoint)
	if err != nil {
		return nil, fmt.Errorf("解析 xusou 搜索地址失败: %w", err)
	}
	values := parsed.Query()
	values.Set("title", query)
	values.Set("is_type", strconv.Itoa(isType))
	parsed.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建 xusou 搜索请求失败: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; URLDB/1.5; +https://github.com/ctwj/urldb)")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xusou 搜索请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("xusou 搜索服务返回 HTTP %d", resp.StatusCode)
	}

	items := make([]XusouSearchResult, 0, 32)
	seen := make(map[string]struct{})
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 8<<20))
	scanner.Buffer(make([]byte, 4<<10), maxXusouEventSize)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if strings.HasPrefix(data, "[DONE]") {
			break
		}

		var item XusouSearchResult
		if err := json.Unmarshal([]byte(data), &item); err != nil {
			// Upstream occasionally emits a non-JSON status event; it is not a
			// result and should not make an otherwise valid stream fail.
			continue
		}
		item.Title = NormalizeMelostText(item.Title, 240)
		item.URL = strings.TrimSpace(item.URL)
		item.LineName = NormalizeMelostText(item.LineName, 80)
		if item.Title == "" || item.URL == "" {
			continue
		}
		if item.IsType == 0 {
			item.IsType = isType
		}
		key := item.URL + "\x00" + item.Title
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 xusou 搜索结果失败: %w", err)
	}
	return items, nil
}

// SearchVideo fetches both supported video sources concurrently. Results are
// returned in a deterministic source order (夸克 first, 迅雷 second), even if
// one upstream responds sooner than the other.
func (x *XusouClient) SearchVideo(ctx context.Context, query string) ([]XusouSearchResult, error) {
	type result struct {
		items []XusouSearchResult
		err   error
	}
	results := make([]result, 2)
	var waitGroup sync.WaitGroup
	waitGroup.Add(2)
	for index, isType := range []int{0, 4} {
		go func(index, isType int) {
			defer waitGroup.Done()
			results[index].items, results[index].err = x.Search(ctx, query, isType)
		}(index, isType)
	}
	waitGroup.Wait()

	var firstErr error
	items := make([]XusouSearchResult, 0, len(results[0].items)+len(results[1].items))
	seen := make(map[string]struct{})
	for _, result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		for _, item := range result.items {
			key := item.URL + "\x00" + item.Title
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			items = append(items, item)
		}
	}
	if len(items) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return items, nil
}

// Resolve requests the encoded URL only after a visitor clicks “获取链接”.
func (x *XusouClient) Resolve(ctx context.Context, encodedURL, title string) (string, error) {
	if strings.TrimSpace(x.resolveURL) == "" {
		return "", fmt.Errorf("xusou 分享地址未配置")
	}
	payload, err := json.Marshal(map[string]string{
		"url":   url.QueryEscape(strings.TrimSpace(encodedURL)),
		"title": NormalizeMelostText(title, 240),
	})
	if err != nil {
		return "", fmt.Errorf("构造 xusou 分享请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, x.resolveURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("创建 xusou 分享请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Referer", "https://xusou.cn/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; URLDB/1.5; +https://github.com/ctwj/urldb)")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("xusou 分享请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("xusou 分享服务返回 HTTP %d", resp.StatusCode)
	}
	var upstream xusouResolveResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256<<10)).Decode(&upstream); err != nil {
		return "", fmt.Errorf("解析 xusou 分享结果失败: %w", err)
	}
	if upstream.Code != http.StatusOK || strings.TrimSpace(upstream.Data.URL) == "" {
		message := strings.TrimSpace(upstream.Message)
		if message == "" {
			message = "xusou 未返回可用分享链接"
		}
		return "", fmt.Errorf("xusou 分享失败: %s", message)
	}
	return strings.TrimSpace(upstream.Data.URL), nil
}

// XusouResultID creates a stable, opaque document id for staging results.
func XusouResultID(item XusouSearchResult) string {
	digest := sha256.Sum256([]byte(item.URL + "\x00" + item.Title))
	return "xusou-" + hex.EncodeToString(digest[:])[:32]
}

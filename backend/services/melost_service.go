package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"io"
	"net/http"
	"strings"
	"time"

	nethtml "golang.org/x/net/html"
)

const (
	defaultMelostEndpoint = "https://melost.cn/v1/search/disk"
	maxMelostBodySize     = 4 << 20
)

// MelostClient wraps the fixed disk-search endpoint. Keeping the remote URL
// server-side avoids exposing the application as a general-purpose proxy.
type MelostClient struct {
	endpoint   string
	httpClient *http.Client
}

type MelostSearchParams struct {
	Query string
	Type  string
	Page  int
	Size  int
}

type MelostSearchResult struct {
	DocID      string   `json:"doc_id"`
	DiskName   string   `json:"disk_name"`
	DiskType   string   `json:"disk_type"`
	Link       string   `json:"link"`
	DiskPass   string   `json:"disk_pass"`
	Files      string   `json:"files"`
	Tags       []string `json:"tags"`
	SharedTime string   `json:"shared_time"`
	ShareUser  string   `json:"share_user"`
	Size       int64    `json:"size"`
}

type MelostSearchResponse struct {
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
	Took     int64                `json:"took"`
	Items    []MelostSearchResult `json:"items"`
}

type melostSearchRequest struct {
	Query        string          `json:"q"`
	Type         string          `json:"type"`
	Page         int             `json:"page"`
	Size         int             `json:"size"`
	ShareTime    string          `json:"share_time"`
	ShareYear    string          `json:"share_year"`
	Format       []string        `json:"format"`
	Exact        bool            `json:"exact"`
	Order        string          `json:"order"`
	SearchTicket string          `json:"search_ticket"`
	ExcludeUser  []string        `json:"exclude_user"`
	UserDistinct bool            `json:"user_distinct"`
	AdvParams    melostAdvParams `json:"adv_params"`
}

type melostAdvParams struct {
	WechatPassword string `json:"wechat_pwd"`
	SearchCode     string `json:"search_code"`
	Platform       string `json:"platform"`
	Fingerprint    string `json:"fp_data"`
}

type melostAPIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Total   int64              `json:"total"`
		PerSize int                `json:"per_size"`
		Took    int64              `json:"took"`
		List    []melostDiskRecord `json:"list"`
	} `json:"data"`
}

type melostDiskRecord struct {
	DocID      string          `json:"doc_id"`
	DiskName   string          `json:"disk_name"`
	DiskType   string          `json:"disk_type"`
	Link       string          `json:"link"`
	DiskPass   string          `json:"disk_pass"`
	Files      string          `json:"files"`
	Tags       json.RawMessage `json:"tags"`
	SharedTime string          `json:"shared_time"`
	ShareUser  string          `json:"share_user"`
	Size       int64           `json:"size"`
	Enabled    bool            `json:"enabled"`
}

func NewMelostClient() *MelostClient {
	return &MelostClient{
		endpoint: defaultMelostEndpoint,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// NewMelostClientWithEndpoint is primarily useful for isolated integration tests.
func NewMelostClientWithEndpoint(endpoint string, client *http.Client) *MelostClient {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &MelostClient{endpoint: endpoint, httpClient: client}
}

func (m *MelostClient) Search(ctx context.Context, params MelostSearchParams) (*MelostSearchResponse, error) {
	payload := melostSearchRequest{
		Query:       params.Query,
		Type:        params.Type,
		Page:        params.Page,
		Size:        params.Size,
		Format:      []string{},
		ExcludeUser: []string{},
		AdvParams: melostAdvParams{
			Platform: "pc",
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("构造 melost 搜索请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 melost 搜索请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://melost.cn")
	req.Header.Set("Referer", "https://melost.cn/search")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; URLDB/1.5; +https://github.com/ctwj/urldb)")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("melost 搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("melost 搜索服务返回 HTTP %d", resp.StatusCode)
	}

	var upstream melostAPIResponse
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxMelostBodySize))
	if err := decoder.Decode(&upstream); err != nil {
		return nil, fmt.Errorf("解析 melost 搜索结果失败: %w", err)
	}

	// melost uses 417 as an application-level empty-result response.
	if upstream.Code == 417 {
		return &MelostSearchResponse{
			Page:     params.Page,
			PageSize: params.Size,
			Items:    []MelostSearchResult{},
		}, nil
	}
	if upstream.Code != http.StatusOK {
		message := strings.TrimSpace(upstream.Msg)
		if message == "" {
			message = fmt.Sprintf("上游错误码 %d", upstream.Code)
		}
		return nil, fmt.Errorf("melost 搜索失败: %s", message)
	}

	items := make([]MelostSearchResult, 0, len(upstream.Data.List))
	for _, record := range upstream.Data.List {
		if !record.Enabled || strings.TrimSpace(record.Link) == "" {
			continue
		}
		items = append(items, MelostSearchResult{
			DocID:      truncateRunes(plainText(record.DocID), 128),
			DiskName:   truncateRunes(plainText(record.DiskName), 255),
			DiskType:   truncateRunes(strings.ToUpper(plainText(record.DiskType)), 32),
			Link:       truncateRunes(strings.TrimSpace(record.Link), 2048),
			DiskPass:   truncateRunes(plainText(record.DiskPass), 64),
			Files:      truncateRunes(plainText(record.Files), 2000),
			Tags:       normalizeMelostTags(record.Tags),
			SharedTime: truncateRunes(plainText(record.SharedTime), 64),
			ShareUser:  truncateRunes(plainText(record.ShareUser), 100),
			Size:       record.Size,
		})
	}

	return &MelostSearchResponse{
		Total:    upstream.Data.Total,
		Page:     params.Page,
		PageSize: params.Size,
		Took:     upstream.Data.Took,
		Items:    items,
	}, nil
}

func plainText(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	document, err := nethtml.Parse(strings.NewReader("<div>" + value + "</div>"))
	if err != nil {
		return strings.TrimSpace(stdhtml.UnescapeString(value))
	}

	var builder strings.Builder
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.TextNode {
			builder.WriteString(node.Data)
			builder.WriteByte(' ')
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return strings.Join(strings.Fields(stdhtml.UnescapeString(builder.String())), " ")
}

// NormalizeMelostText removes upstream markup before values are persisted or displayed.
func NormalizeMelostText(value string, limit int) string {
	return truncateRunes(plainText(value), limit)
}

func normalizeMelostTags(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}
	}
	var tags []string
	if err := json.Unmarshal(raw, &tags); err != nil {
		return []string{}
	}
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		if clean := truncateRunes(plainText(tag), 64); clean != "" {
			result = append(result, clean)
		}
	}
	return result
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

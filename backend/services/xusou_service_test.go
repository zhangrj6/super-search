package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestXusouClientSearchParsesSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("title") != "王国" || r.URL.Query().Get("is_type") != "0" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Fatalf("accept = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, ":心跳\n\ndata: {\"title\":\"<em>王国</em>\",\"url\":\"encoded-1\",\"is_type\":0,\"line_name\":\"聚合\"}\n\ndata: {\"title\":\"王国\",\"url\":\"encoded-1\",\"is_type\":0,\"line_name\":\"重复线路\"}\n\ndata: 线路完成\n\ndata: [DONE] 共返回 2 个结果\n")
	}))
	defer server.Close()

	client := NewXusouClientWithEndpoint(server.URL, server.Client())
	items, err := client.Search(context.Background(), "王国", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(items) != 1 || items[0].Title != "王国" || items[0].URL != "encoded-1" || items[0].LineName != "聚合" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestXusouClientSearchVideoConcatenatesSources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isType := r.URL.Query().Get("is_type")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"title\":\"source-"+isType+"\",\"url\":\"url-"+isType+"\",\"is_type\":"+isType+"}\n\ndata: [DONE]\n")
	}))
	defer server.Close()

	client := NewXusouClientWithEndpoint(server.URL, server.Client())
	items, err := client.SearchVideo(context.Background(), "王国")
	if err != nil {
		t.Fatalf("SearchVideo() error = %v", err)
	}
	if len(items) != 2 || items[0].IsType != 0 || items[1].IsType != 4 {
		t.Fatalf("unexpected merged items: %#v", items)
	}
}

func TestXusouClientResolve(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["url"] != url.QueryEscape("encoded+/=") || payload["title"] != "标题" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":200,"message":"临时资源获取成功","data":{"url":"https://pan.quark.cn/s/example?pwd=abcd"}}`)
	}))
	defer server.Close()

	client := NewXusouClientWithEndpoints(server.URL+"/search", server.URL+"/save", server.Client())
	link, err := client.Resolve(context.Background(), "encoded+/=", "标题")
	if err != nil || !strings.HasPrefix(link, "https://pan.quark.cn/") {
		t.Fatalf("Resolve() = %q, %v", link, err)
	}
}

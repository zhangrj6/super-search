package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMelostClientSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/search/disk" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var request melostSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Query != "蝙蝠侠" || request.Page != 2 || request.Size != 10 {
			t.Fatalf("unexpected request: %+v", request)
		}
		if request.AdvParams.Platform != "pc" || request.Format == nil || request.ExcludeUser == nil {
			t.Fatalf("missing canonical melost fields: %+v", request)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "code":200,
            "msg":"请求成功",
            "data":{"total":1,"per_size":10,"took":12,"list":[{
              "doc_id":"doc-1",
              "disk_name":"<em>蝙蝠侠</em><script>alert(1)</script>",
              "disk_type":"quark",
              "link":"https://pan.quark.cn/s/abc123",
              "disk_pass":"",
              "files":"<em>电影</em> 合集",
              "tags":["4K"],
              "shared_time":"2026-08-09 08:42:37",
              "share_user":"测试用户",
              "enabled":true,
              "size":0
            }]}
          }`))
	}))
	defer server.Close()

	client := NewMelostClientWithEndpoint(server.URL+"/v1/search/disk", server.Client())
	result, err := client.Search(context.Background(), MelostSearchParams{
		Query: "蝙蝠侠",
		Page:  2,
		Size:  10,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 1 || result.Took != 12 || len(result.Items) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	item := result.Items[0]
	if strings.Contains(item.DiskName, "<") || item.DiskName != "蝙蝠侠 alert(1)" {
		t.Fatalf("disk name was not normalized: %q", item.DiskName)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "4K" {
		t.Fatalf("unexpected tags: %#v", item.Tags)
	}
}

func TestMelostClientSearchTreats417AsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":417,"msg":"没有搜索结果","data":{"total":0,"list":null}}`))
	}))
	defer server.Close()

	client := NewMelostClientWithEndpoint(server.URL, server.Client())
	result, err := client.Search(context.Background(), MelostSearchParams{Query: "不存在", Page: 1, Size: 20})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
}

func TestMelostClientSearchRejectsUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewMelostClientWithEndpoint(server.URL, server.Client())
	_, err := client.Search(context.Background(), MelostSearchParams{Query: "test", Page: 1, Size: 20})
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("expected HTTP 503 error, got %v", err)
	}
}

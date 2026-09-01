package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestQuanpanClientSearchObtainsAndCachesToken(t *testing.T) {
	var mu sync.Mutex
	tokenCalls := 0
	searchCalls := 0

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/api/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("token method = %s", r.Method)
		}
		if r.Header.Get("Referer") != server.URL+"/" || r.Header.Get("Sec-Fetch-Site") != "same-origin" {
			t.Fatalf("unexpected token headers: %#v", r.Header)
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "Chrome/148") {
			t.Fatalf("unexpected user agent: %q", r.Header.Get("User-Agent"))
		}
		mu.Lock()
		tokenCalls++
		mu.Unlock()
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"token":"future.signature","ttl_ms":900000}}`)
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("search method = %s", r.Method)
		}
		if r.Header.Get("X-QP-K") != "future.signature" {
			t.Fatalf("X-QP-K = %q", r.Header.Get("X-QP-K"))
		}
		if r.Header.Get("Origin") != server.URL || !strings.HasPrefix(r.Header.Get("Referer"), server.URL+"/?q=") {
			t.Fatalf("unexpected search headers: %#v", r.Header)
		}
		var payload struct {
			Keyword    string   `json:"kw"`
			Result     string   `json:"res"`
			Source     string   `json:"src"`
			CloudTypes []string `json:"cloud_types"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode search request: %v", err)
		}
		if payload.Keyword != "早春晴朗" || payload.Result != "quark,xunlei" || payload.Source != "all" {
			t.Fatalf("unexpected search request: %#v", payload)
		}
		if len(payload.CloudTypes) != 1 || payload.CloudTypes[0] != "quark" {
			t.Fatalf("cloud_types = %#v", payload.CloudTypes)
		}
		mu.Lock()
		searchCalls++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"code":0,
			"message":"success",
			"data":{
				"total":4,
				"merged_by_type":{
					"quark":[
						{"url":"https://pan.quark.cn/s/abc","password":" p123 ","note":"<b>早春晴朗 4K</b>","datetime":"2026-09-01T08:00:00+08:00"},
						{"url":"https://pan.quark.cn/s/abc","password":"","note":"重复","datetime":""},
						{"url":"https://pan.quark.cn/s/empty","password":"","note":"","datetime":""}
					],
					"xunlei":[{"url":"https://pan.xunlei.com/s/other","password":"","note":"其它网盘","datetime":""}]
				}
			}
		}`)
	})

	client := NewQuanpanClientWithEndpoints(server.URL+"/api/auth/token", server.URL+"/api/search", server.Client())
	for call := 0; call < 2; call++ {
		items, err := client.Search(context.Background(), " 早春晴朗 ", "QUARK")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("items = %#v", items)
		}
		if items[0].Note != "早春晴朗 4K" || items[0].Password != "p123" || items[0].URL != "https://pan.quark.cn/s/abc?pwd=p123" {
			t.Fatalf("unexpected normalized item: %#v", items[0])
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if tokenCalls != 1 || searchCalls != 2 {
		t.Fatalf("token calls = %d, search calls = %d", tokenCalls, searchCalls)
	}
}

func TestQuanpanClientRefreshesTokenAfterForbidden(t *testing.T) {
	var mu sync.Mutex
	tokenCalls := 0
	searchCalls := 0

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		tokenCalls++
		token := fmt.Sprintf("token-%d", tokenCalls)
		mu.Unlock()
		_, _ = fmt.Fprintf(w, `{"code":0,"data":{"token":%q,"ttl_ms":900000}}`, token)
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		searchCalls++
		call := searchCalls
		mu.Unlock()
		if call == 1 {
			if r.Header.Get("X-QP-K") != "token-1" {
				t.Fatalf("first token = %q", r.Header.Get("X-QP-K"))
			}
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Header.Get("X-QP-K") != "token-2" {
			t.Fatalf("refreshed token = %q", r.Header.Get("X-QP-K"))
		}
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"total":1,"merged_by_type":{"xunlei":[{"url":"https://pan.xunlei.com/s/abc","password":"1234","note":"视频","datetime":""}]}}}`)
	})

	client := NewQuanpanClientWithEndpoints(server.URL+"/token", server.URL+"/search", server.Client())
	items, err := client.Search(context.Background(), "视频", "xunlei")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(items) != 1 || items[0].Password != "1234" {
		t.Fatalf("items = %#v", items)
	}
	mu.Lock()
	defer mu.Unlock()
	if tokenCalls != 2 || searchCalls != 2 {
		t.Fatalf("token calls = %d, search calls = %d", tokenCalls, searchCalls)
	}
}

func TestQuanpanClientRejectsUnsupportedProvider(t *testing.T) {
	client := NewQuanpanClientWithEndpoints("", "", nil)
	if _, err := client.Search(context.Background(), "视频", "baidu"); err == nil {
		t.Fatal("Search() should reject unsupported provider")
	}
}

func TestQuanpanURLWithPassword(t *testing.T) {
	tests := []struct {
		name     string
		rawURL   string
		password string
		want     string
	}{
		{name: "append", rawURL: "https://pan.quark.cn/s/abc", password: "a+b", want: "https://pan.quark.cn/s/abc?pwd=a%2Bb"},
		{name: "keep existing", rawURL: "https://pan.xunlei.com/s/abc?pwd=old#", password: "new", want: "https://pan.xunlei.com/s/abc?pwd=old#"},
		{name: "empty password", rawURL: "https://pan.quark.cn/s/abc", password: "", want: "https://pan.quark.cn/s/abc"},
		{name: "malformed URL", rawURL: "://bad", password: "1234", want: "://bad"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := quanpanURLWithPassword(test.rawURL, test.password); got != test.want {
				t.Fatalf("quanpanURLWithPassword(%q, %q) = %q, want %q", test.rawURL, test.password, got, test.want)
			}
		})
	}
}

func TestQuanpanResultIDIsStable(t *testing.T) {
	item := QuanpanSearchResult{URL: "https://pan.quark.cn/s/abc", Note: "视频"}
	first := QuanpanResultID(item)
	second := QuanpanResultID(item)
	if first != second || !strings.HasPrefix(first, "quanpan-") || len(first) != len("quanpan-")+32 {
		t.Fatalf("unexpected result id: %q, %q", first, second)
	}
}

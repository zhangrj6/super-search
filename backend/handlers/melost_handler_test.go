package handlers

import (
	"strings"
	"testing"
)

func TestTransferServiceForURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		serviceType string
		supported   bool
	}{
		{name: "quark", url: "https://pan.quark.cn/s/abc123", serviceType: "quark", supported: true},
		{name: "alipan", url: "https://www.alipan.com/s/abc123", serviceType: "alipan", supported: true},
		{name: "baidu", url: "https://pan.baidu.com/s/abc_123", serviceType: "baidu", supported: true},
		{name: "reject lookalike host", url: "https://pan.quark.cn.example.com/s/abc123", supported: false},
		{name: "reject http", url: "http://pan.quark.cn/s/abc123", supported: false},
		{name: "reject credentials", url: "https://user@pan.quark.cn/s/abc123", supported: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceType, _, supported := transferServiceForURL(tt.url)
			if supported != tt.supported || serviceType != tt.serviceType {
				t.Fatalf("transferServiceForURL(%q) = (%q, %v), want (%q, %v)", tt.url, serviceType, supported, tt.serviceType, tt.supported)
			}
		})
	}
}

func TestIsAllowedMelostType(t *testing.T) {
	for _, value := range []string{"QUARK", "XUNLEI"} {
		if !isAllowedMelostType(value) {
			t.Fatalf("expected type %q to be allowed", value)
		}
	}
	for _, value := range []string{"", "BDY", "ALY", "UC", "MAGNET"} {
		if isAllowedMelostType(value) {
			t.Fatalf("expected type %q to be rejected", value)
		}
	}
}

func TestQuanpanProviderForType(t *testing.T) {
	tests := []struct {
		provider string
		wantType string
		wantOK   bool
	}{
		{provider: "QUARK", wantType: "quark", wantOK: true},
		{provider: "quark", wantType: "quark", wantOK: true},
		{provider: "XUNLEI", wantType: "xunlei", wantOK: true},
		{provider: "BDY", wantType: "", wantOK: false},
	}
	for _, test := range tests {
		gotType, gotOK := quanpanProviderForType(test.provider)
		if gotType != test.wantType || gotOK != test.wantOK {
			t.Fatalf("quanpanProviderForType(%q) = (%q, %v), want (%q, %v)", test.provider, gotType, gotOK, test.wantType, test.wantOK)
		}
	}
}

func TestBuildMelostDescription(t *testing.T) {
	description := buildMelostDescription(melostStageRequest{
		Files:      "<b>电影合集</b>",
		Tags:       []string{" 影视 ", "<i>高清</i>"},
		SharedTime: "2026-08-28",
		DiskPass:   "a1b2",
	})

	for _, expected := range []string{"电影合集", "标签：影视、高清", "分享时间：2026-08-28", "提取码：a1b2"} {
		if !strings.Contains(description, expected) {
			t.Fatalf("description %q does not contain %q", description, expected)
		}
	}
	if strings.Contains(description, "<b>") || strings.Contains(description, "<i>") {
		t.Fatalf("description should not contain upstream HTML: %q", description)
	}
}

func TestFormatMelostSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{size: 0, want: ""},
		{size: 512, want: "512 B"},
		{size: 1536, want: "1.5 KB"},
		{size: 2 * 1024 * 1024 * 1024, want: "2.0 GB"},
	}

	for _, test := range tests {
		if got := formatMelostSize(test.size); got != test.want {
			t.Fatalf("formatMelostSize(%d) = %q, want %q", test.size, got, test.want)
		}
	}
}

package pan

import "testing"

func TestCheckAdKeywordsInFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		keywords []string
		want     bool
	}{
		{name: "builtin keyword", filename: "官方推广说明.txt", want: true},
		{name: "resource search pdf", filename: "影盘社资源搜索(2).pdf", want: true},
		{name: "resource search apk", filename: "影盘社资源搜索(2).apk", want: true},
		{name: "markdown url", filename: "更多免费资源【[www.melost(1).cn】影盘社](http://www.melost\\(1\\).cn】影盘社)", want: true},
		{name: "bare versioned domain", filename: "更多免费资源 melost(1).cn.pdf", want: true},
		{name: "separator bypass", filename: "微-信联系.txt", want: true},
		{name: "full width", filename: "ＱＱ群福利.txt", want: true},
		{name: "ordinary apk is not an ad", filename: "OCR文字识别工具.apk", want: false},
		{name: "content keyword is not enough", filename: "广告片案例分析.mp4", want: false},
		{name: "custom keyword", filename: "某来源_notice.mkv", keywords: []string{"notice"}, want: true},
		{name: "normal resource", filename: "Foundation.S02E01.1080p.mkv", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkAdKeywordsInFilename(tt.filename, tt.keywords); got != tt.want {
				t.Fatalf("checkAdKeywordsInFilename(%q, %v) = %v, want %v", tt.filename, tt.keywords, got, tt.want)
			}
		})
	}
}

func TestIsPanDirectory(t *testing.T) {
	if !isPanDirectory(map[string]interface{}{"dir": true, "file_type": float64(0)}) {
		t.Fatal("dir=true should identify a directory")
	}
	if isPanDirectory(map[string]interface{}{"file": true, "file_type": float64(1)}) {
		t.Fatal("file=true should identify a regular file")
	}
	if !isPanDirectory(map[string]interface{}{"file_type": float64(0)}) {
		t.Fatal("legacy file_type=0 should identify a directory")
	}
	if isPanDirectory(map[string]interface{}{"file_type": float64(1)}) {
		t.Fatal("legacy file_type=1 should identify a regular file")
	}
}

func TestTransferPromoFolderName(t *testing.T) {
	const want = "更多免费资源 52juyou.com 聚优盘"
	if transferPromoFolderName != want {
		t.Fatalf("transferPromoFolderName = %q, want %q", transferPromoFolderName, want)
	}
}

func TestSplitAndAppendPanFIDs(t *testing.T) {
	fids := splitPanFIDs(" first,second, first, ")
	if len(fids) != 2 || fids[0] != "first" || fids[1] != "second" {
		t.Fatalf("splitPanFIDs() = %v", fids)
	}
	fids, err := appendPanFID(fids, "promo")
	if err != nil {
		t.Fatalf("appendPanFID() unexpected error: %v", err)
	}
	if len(fids) != 3 || fids[2] != "promo" {
		t.Fatalf("appendPanFID() = %v", fids)
	}
}

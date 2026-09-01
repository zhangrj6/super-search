package pan

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/ctwj/urldb/db"
	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
)

const transferPromoFolderName = "更多免费资源 52juyou.com 聚优盘"

var transferPromoFolderMu sync.Mutex

// defaultAdKeywords 是始终启用的高置信度规则。管理员可以通过 ad_keywords
// 追加业务相关词，但不能关闭这些基础规则，避免配置为空时广告过滤失效。
var defaultAdKeywords = []string{
	"推广", "商务合作", "合作联系", "广告合作", "广告联系", "广告位", "加群", "入群", "会员群", "资源群",
	"微信群", "QQ群", "q群", "公众号", "微信", "加微", "V信", "威信", "扫码", "二维码",
	"私聊", "优惠券", "返利", "抽奖", "特价", "折扣", "限时", "免费领取", "网盘推广",
	"资源合集", "资源汇总", "影视资源", "免费资源", "免费全网资源", "更多资源", "获取更多",
	"网盘搜索", "资源搜索", "搜索资源", "搜索小程序", "资源小程序", "小程序", "影盘社", "日入", "网盘玩法", "妙妙屋", "胖狗资源",
	"dy8.xyz", "kkdm", "yuntv.net", "xfzyk.cn", ".url", "IMG_",
}

var (
	// URL、域名和即时通讯账号是文件名中最稳定的广告信号。
	adURLPattern = regexp.MustCompile(`(?i)(?:https?://|www\.)[^\s]+`)
	// 兼容下载器生成的 melost(1).cn、melost\(1\).cn 这类重名文件。
	adBareDomainPattern = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(?:[a-z0-9-]+(?:\\?\(\d+\\?\))?\.)+(?:com|cn|net|org|cc|xyz|top|vip|site|link|app|me|tv|io)(?:$|[^a-z0-9])`)
	adContactPattern    = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])(qq|vx|wx)[-_ ]?\d{4,}(?:$|[^a-z0-9])`)
)

// 本文件提供网盘"广告清理 / 自定义广告插入"的共享工具函数。
//
// 背景：夸克、UC 等网盘共用同一套广告规则，避免不同平台的转存流程出现规则漂移。
//
// 相关系统配置项（db/entity）：
//   - ConfigKeyAdKeywords：广告关键词（命中则删除来源广告文件）
//   - ConfigKeyAutoInsertAd：自动插入广告（转存后随机插入一条自定义广告）
//
// 依赖的包级缓存变量（configRefreshChan / systemConfigOnce / systemConfigRepo）
// 声明于 quark_pan.go，同包内直接引用。

// getAdSystemConfigValue 读取系统配置值，并在收到缓存刷新信号时清空缓存。
func getAdSystemConfigValue(key string) (string, error) {
	// 检查是否需要刷新缓存
	select {
	case <-configRefreshChan:
		systemConfigOnce.Do(func() {
			systemConfigRepo = repo.NewSystemConfigRepository(db.DB)
		})
		systemConfigRepo.ClearConfigCache()
	default:
		// 没有刷新信号，继续使用缓存
	}

	systemConfigOnce.Do(func() {
		systemConfigRepo = repo.NewSystemConfigRepository(db.DB)
	})
	return systemConfigRepo.GetConfigValue(key)
}

// splitAdKeywords 按中英文逗号分割关键词。
func splitAdKeywords(keywordsStr string) []string {
	if keywordsStr == "" {
		return []string{}
	}
	re := regexp.MustCompile(`[,，]`)
	parts := re.Split(keywordsStr, -1)

	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// splitAdURLs 按换行符分割广告 URL 列表。
func splitAdURLs(autoInsertAdStr string) []string {
	if autoInsertAdStr == "" {
		return []string{}
	}
	lines := strings.Split(autoInsertAdStr, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// checkAdKeywordsInFilename 检查文件名是否包含任一关键词（大小写不敏感）。
func checkAdKeywordsInFilename(filename string, keywords []string) bool {
	if strings.TrimSpace(filename) == "" {
		return false
	}

	normalizedFilename := normalizeAdText(filename)
	compactFilename := compactAdText(normalizedFilename)
	if adURLPattern.MatchString(normalizedFilename) || adBareDomainPattern.MatchString(normalizedFilename) || adContactPattern.MatchString(normalizedFilename) {
		log.Printf("文件 %s 命中广告特征规则", filename)
		return true
	}

	for _, keyword := range defaultAdKeywords {
		if matchesAdKeyword(normalizedFilename, compactFilename, keyword) {
			log.Printf("文件 %s 包含广告关键词: %s", filename, keyword)
			return true
		}
	}
	for _, keyword := range keywords {
		if matchesAdKeyword(normalizedFilename, compactFilename, keyword) {
			log.Printf("文件 %s 包含广告关键词: %s", filename, keyword)
			return true
		}
	}
	return false
}

func matchesAdKeyword(normalizedFilename, compactFilename, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	normalizedKeyword := normalizeAdText(keyword)
	// 以点号开头的规则（例如 .url）只做原文匹配，避免把普通单词中的 url 误判。
	if strings.HasPrefix(normalizedKeyword, ".") {
		return strings.Contains(normalizedFilename, normalizedKeyword)
	}
	return strings.Contains(normalizedFilename, normalizedKeyword) ||
		strings.Contains(compactFilename, compactAdText(normalizedKeyword))
}

// normalizeAdText 统一全半角、大小写并移除不可见控制字符。
func normalizeAdText(value string) string {
	value = norm.NFKC.String(value)
	value = strings.ToLower(value)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u200b' || r == '\u200c' || r == '\u200d' || r == '\ufeff' {
			return -1
		}
		return r
	}, value)
}

// compactAdText 删除空白、标点和符号，用于识别通过分隔符拆开的广告词。
func compactAdText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value)
}

// isPanDirectory 兼容夸克/UC 接口的目录标记。
func isPanDirectory(file map[string]interface{}) bool {
	if dir, ok := file["dir"].(bool); ok {
		return dir
	}
	if isFile, ok := file["file"].(bool); ok {
		return !isFile
	}
	if fileType, ok := file["file_type"].(float64); ok {
		return fileType == 0
	}
	if fileType, ok := file["file_type"].(int); ok {
		return fileType == 0
	}
	return false
}

// splitPanFIDs 兼容历史记录中逗号拼接的多个文件 ID。
func splitPanFIDs(value string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		fid := strings.TrimSpace(part)
		if fid == "" {
			continue
		}
		if _, exists := seen[fid]; exists {
			continue
		}
		seen[fid] = struct{}{}
		result = append(result, fid)
	}
	return result
}

func appendPanFID(fidList []string, fid string) ([]string, error) {
	fid = strings.TrimSpace(fid)
	if fid == "" {
		return nil, fmt.Errorf("文件夹 FID 为空")
	}
	result := append([]string(nil), fidList...)
	for _, existing := range result {
		if existing == fid {
			return result, nil
		}
	}
	return append(result, fid), nil
}

// containsAdKeywords 检查文件名是否命中系统配置的广告关键词。
func containsAdKeywords(filename string) bool {
	if db.DB == nil {
		return checkAdKeywordsInFilename(filename, nil)
	}
	adKeywordsStr, err := getAdSystemConfigValue(entity.ConfigKeyAdKeywords)
	if err != nil {
		log.Printf("获取广告关键词配置失败，使用内置规则: %v", err)
		return checkAdKeywordsInFilename(filename, nil)
	}
	return checkAdKeywordsInFilename(filename, splitAdKeywords(adKeywordsStr))
}

// extractAdFileIDs 从广告 URL 列表中提取分享 ID（复用同包 ExtractShareId）。
func extractAdFileIDs(adURLs []string) []string {
	var result []string
	for _, url := range adURLs {
		shareID, _ := ExtractShareId(url)
		if shareID != "" {
			result = append(result, shareID)
		}
	}
	return result
}

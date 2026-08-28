package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	pan "github.com/ctwj/urldb/common"
	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
	"github.com/ctwj/urldb/services"
	"github.com/ctwj/urldb/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MelostHandler struct {
	repoMgr *repo.RepositoryManager
	client  *services.MelostClient
	stageMu sync.Mutex
}

type melostSearchRequest struct {
	Query string `json:"q" binding:"required"`
	Type  string `json:"type"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
}

type melostStageRequest struct {
	DocID      string   `json:"doc_id"`
	Title      string   `json:"title" binding:"required"`
	Link       string   `json:"link" binding:"required"`
	DiskType   string   `json:"disk_type"`
	DiskPass   string   `json:"disk_pass"`
	Files      string   `json:"files"`
	Tags       []string `json:"tags"`
	SharedTime string   `json:"shared_time"`
	ShareUser  string   `json:"share_user"`
	Size       int64    `json:"size"`
}

type melostSearchItemResponse struct {
	services.MelostSearchResult
	CanStage     bool   `json:"can_stage"`
	StageMessage string `json:"stage_message,omitempty"`
}

func NewMelostHandler(repoMgr *repo.RepositoryManager) *MelostHandler {
	return &MelostHandler{
		repoMgr: repoMgr,
		client:  services.NewMelostClient(),
	}
}

func (h *MelostHandler) Search(c *gin.Context) {
	var request melostSearchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		ErrorResponse(c, "请输入搜索关键词", http.StatusBadRequest)
		return
	}

	request.Query = strings.TrimSpace(request.Query)
	if request.Query == "" || utf8.RuneCountInString(request.Query) > 100 {
		ErrorResponse(c, "搜索关键词长度应为 1 到 100 个字符", http.StatusBadRequest)
		return
	}
	request.Type = strings.ToUpper(strings.TrimSpace(request.Type))
	if !isAllowedMelostType(request.Type) {
		ErrorResponse(c, "不支持的网盘类型", http.StatusBadRequest)
		return
	}
	if request.Page < 1 {
		request.Page = 1
	}
	if request.Page > 500 {
		request.Page = 500
	}
	if request.Size < 1 || request.Size > 20 {
		request.Size = 20
	}

	result, err := h.client.Search(c.Request.Context(), services.MelostSearchParams{
		Query: request.Query,
		Type:  request.Type,
		Page:  request.Page,
		Size:  request.Size,
	})
	if err != nil {
		utils.Error("melost 搜索失败 query=%q: %v", request.Query, err)
		ErrorResponse(c, "站外搜索暂时不可用，请稍后重试", http.StatusBadGateway)
		return
	}

	items := make([]melostSearchItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		_, _, supported := transferServiceForURL(item.Link)
		response := melostSearchItemResponse{
			MelostSearchResult: item,
			CanStage:           supported,
		}
		if !supported {
			response.StageMessage = "该链接类型暂不支持转存"
		}
		items = append(items, response)
	}

	SuccessResponse(c, gin.H{
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
		"took":      result.Took,
		"items":     items,
	})
}

// StageResource saves a melost result locally without accessing a cloud-drive account.
func (h *MelostHandler) StageResource(c *gin.Context) {
	var request melostStageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		ErrorResponse(c, "资源参数不完整", http.StatusBadRequest)
		return
	}

	request.Title = services.NormalizeMelostText(request.Title, 240)
	request.DocID = services.NormalizeMelostText(request.DocID, 128)
	request.ShareUser = services.NormalizeMelostText(request.ShareUser, 100)
	request.Link = strings.TrimSpace(request.Link)
	if request.Title == "" {
		ErrorResponse(c, "资源标题不能为空", http.StatusBadRequest)
		return
	}

	serviceType, _, supported := transferServiceForURL(request.Link)
	if !supported {
		ErrorResponse(c, "该链接类型暂不支持转存", http.StatusBadRequest)
		return
	}
	panID, err := h.repoMgr.PanRepository.FindIdByServiceType(serviceType)
	if err != nil || panID <= 0 {
		utils.Error("melost 资源平台不存在 service=%s: %v", serviceType, err)
		ErrorResponse(c, "系统未配置该网盘平台", http.StatusConflict)
		return
	}

	// Keep lookup and creation atomic within this server process so repeated
	// clicks cannot create duplicate records.
	h.stageMu.Lock()
	defer h.stageMu.Unlock()

	existing, err := h.repoMgr.ResourceRepository.GetByURL(request.Link)
	if err == nil {
		fields := map[string]interface{}{
			"source":      "melost",
			"external_id": request.DocID,
			"is_valid":    true,
			"is_public":   true,
		}
		if strings.TrimSpace(existing.Key) == "" {
			key, keyErr := h.repoMgr.ResourceRepository.GenerateUniqueKey()
			if keyErr != nil {
				ErrorResponse(c, "生成资源详情地址失败", http.StatusInternalServerError)
				return
			}
			existing.Key = key
			fields["key"] = key
		}
		if err := h.repoMgr.ResourceRepository.UpdateFields(existing.ID, fields); err != nil {
			utils.Error("更新 melost 暂存资源失败 id=%d: %v", existing.ID, err)
			ErrorResponse(c, "保存资源失败", http.StatusInternalServerError)
			return
		}
		SuccessResponse(c, gin.H{
			"existing":     true,
			"resource_id":  existing.ID,
			"resource_key": existing.Key,
			"status":       "staged",
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Error("查询 melost 暂存资源失败 url=%s: %v", request.Link, err)
		ErrorResponse(c, "检查资源是否已存在失败", http.StatusInternalServerError)
		return
	}

	key, err := h.repoMgr.ResourceRepository.GenerateUniqueKey()
	if err != nil {
		ErrorResponse(c, "生成资源详情地址失败", http.StatusInternalServerError)
		return
	}
	panIDValue := uint(panID)
	resource := &entity.Resource{
		Title:       request.Title,
		Description: buildMelostDescription(request),
		URL:         request.Link,
		PanID:       &panIDValue,
		SaveURL:     "",
		FileSize:    formatMelostSize(request.Size),
		IsValid:     true,
		IsPublic:    true,
		Author:      request.ShareUser,
		Key:         key,
		Source:      "melost",
		ExternalID:  request.DocID,
	}
	if err := h.repoMgr.ResourceRepository.Create(resource); err != nil {
		utils.Error("创建 melost 暂存资源失败: %v", err)
		ErrorResponse(c, "保存资源失败", http.StatusInternalServerError)
		return
	}

	SuccessResponse(c, gin.H{
		"existing":     false,
		"resource_id":  resource.ID,
		"resource_key": resource.Key,
		"status":       "staged",
	})
}

func buildMelostDescription(request melostStageRequest) string {
	parts := make([]string, 0, 4)
	if files := services.NormalizeMelostText(request.Files, 1000); files != "" {
		parts = append(parts, files)
	}
	if len(request.Tags) > 0 {
		tags := make([]string, 0, len(request.Tags))
		for _, tag := range request.Tags {
			if normalized := services.NormalizeMelostText(tag, 40); normalized != "" {
				tags = append(tags, normalized)
			}
		}
		if len(tags) > 0 {
			parts = append(parts, "标签："+strings.Join(tags, "、"))
		}
	}
	if sharedTime := services.NormalizeMelostText(request.SharedTime, 40); sharedTime != "" {
		parts = append(parts, "分享时间："+sharedTime)
	}
	if diskPass := services.NormalizeMelostText(request.DiskPass, 20); diskPass != "" {
		parts = append(parts, "提取码："+diskPass)
	}
	return strings.Join(parts, "\n")
}

func formatMelostSize(size int64) string {
	if size <= 0 {
		return ""
	}
	const unit = int64(1024)
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}
	divisor := unit
	units := []string{"KB", "MB", "GB", "TB", "PB"}
	unitIndex := 0
	for unitIndex < len(units)-1 && size/divisor >= unit {
		divisor *= unit
		unitIndex++
	}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(divisor), units[unitIndex])
}

func transferServiceForURL(rawURL string) (serviceType, displayName string, supported bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil {
		return "", "", false
	}
	host := strings.ToLower(parsed.Hostname())
	allowedHost := host == "pan.quark.cn" || host == "pan.xunlei.com" || host == "pan.baidu.com" ||
		host == "drive.uc.cn" || host == "fast.uc.cn" || host == "alipan.com" || host == "www.alipan.com" ||
		host == "aliyundrive.com" || host == "www.aliyundrive.com"
	if !allowedHost {
		return "", "", false
	}

	switch pan.ExtractServiceType(rawURL) {
	case pan.Quark:
		return "quark", "夸克网盘", true
	case pan.Alipan:
		return "alipan", "阿里云盘", true
	case pan.BaiduPan:
		return "baidu", "百度网盘", true
	case pan.UC:
		return "uc", "UC网盘", true
	case pan.Xunlei:
		return "xunlei", "迅雷网盘", true
	default:
		return "", "", false
	}
}

func isAllowedMelostType(value string) bool {
	switch value {
	case "", "BDY", "ALY", "QUARK", "XUNLEI", "UC":
		return true
	default:
		return false
	}
}

package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var validTransferOperations = map[string]bool{
	entity.TransferOperationTransfer: true,
	entity.TransferOperationShare:    true,
}

var validTransferStatuses = map[string]bool{
	entity.TransferRecordStatusSucceeded: true,
	entity.TransferRecordStatusFailed:    true,
}

var validTransferCleanupStatuses = map[string]bool{
	entity.TransferCleanupPending: true,
	entity.TransferCleanupCleaned: true,
	entity.TransferCleanupFailed:  true,
}

func GetTransferRecords(c *gin.Context) {
	page, pageErr := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, sizeErr := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageErr != nil || page < 1 || sizeErr != nil || pageSize < 1 || pageSize > 100 {
		ErrorResponse(c, "分页参数无效，page 必须大于 0，page_size 必须在 1-100 之间", http.StatusBadRequest)
		return
	}

	operation := strings.TrimSpace(c.Query("operation"))
	status := strings.TrimSpace(c.Query("status"))
	cleanupStatus := strings.TrimSpace(c.Query("cleanup_status"))
	if operation != "" && !validTransferOperations[operation] {
		ErrorResponse(c, "无效的操作类型", http.StatusBadRequest)
		return
	}
	if status != "" && !validTransferStatuses[status] {
		ErrorResponse(c, "无效的操作状态", http.StatusBadRequest)
		return
	}
	if cleanupStatus != "" && !validTransferCleanupStatuses[cleanupStatus] {
		ErrorResponse(c, "无效的清理状态", http.StatusBadRequest)
		return
	}

	startAt, err := parseTransferRecordDate(c.Query("start_date"), false)
	if err != nil {
		ErrorResponse(c, "开始日期格式无效", http.StatusBadRequest)
		return
	}
	endAt, err := parseTransferRecordDate(c.Query("end_date"), true)
	if err != nil {
		ErrorResponse(c, "结束日期格式无效", http.StatusBadRequest)
		return
	}
	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		ErrorResponse(c, "开始日期不能晚于结束日期", http.StatusBadRequest)
		return
	}

	records, total, err := repoManager.TransferRecordRepository.FindWithFilters(repo.TransferRecordFilter{
		Page:          page,
		Limit:         pageSize,
		Query:         c.Query("query"),
		Operation:     operation,
		Status:        status,
		CleanupStatus: cleanupStatus,
		PanType:       strings.TrimSpace(c.Query("pan_type")),
		StartAt:       startAt,
		EndAt:         endAt,
	})
	if err != nil {
		ErrorResponse(c, "获取转存链路记录失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	SuccessResponse(c, gin.H{
		"records":   records,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetTransferRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		ErrorResponse(c, "无效的记录 ID", http.StatusBadRequest)
		return
	}
	record, err := repoManager.TransferRecordRepository.FindByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ErrorResponse(c, "转存链路记录不存在", http.StatusNotFound)
			return
		}
		ErrorResponse(c, "获取转存链路详情失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	SuccessResponse(c, record)
}

func GetTransferRecordSummary(c *gin.Context) {
	summary, err := repoManager.TransferRecordRepository.GetSummary(time.Now())
	if err != nil {
		ErrorResponse(c, "获取转存链路统计失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	SuccessResponse(c, summary)
}

func parseTransferRecordDate(value string, endOfDay bool) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}

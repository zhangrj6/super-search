package repo

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ctwj/urldb/db/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxTransferCleanupAttempts = 5

type TransferRecordFilter struct {
	Page          int
	Limit         int
	Query         string
	Operation     string
	Status        string
	CleanupStatus string
	PanType       string
	StartAt       *time.Time
	EndAt         *time.Time
}

type TransferRecordRepository interface {
	BaseRepository[entity.TransferRecord]
	CreateRecord(record *entity.TransferRecord) error
	FindWithFilters(filter TransferRecordFilter) ([]entity.TransferRecord, int64, error)
	GetSummary(now time.Time) (*entity.TransferRecordSummary, error)
	FindDueForCleanup(now time.Time, limit int) ([]entity.TransferRecord, error)
	ExtendPendingCleanup(resourceID uint, accountID uint, fileID string, dueAt time.Time) error
	IsFileDueForCleanup(accountID uint, fileID string, now time.Time) (bool, error)
	MarkFileCleaned(accountID uint, fileID string, cleanedAt time.Time) error
	MarkFileCleanupError(accountID uint, fileID string, message string, attemptedAt time.Time) error
	MarkCleanupError(id uint, message string, attemptedAt time.Time) error
}

type TransferRecordRepositoryImpl struct {
	BaseRepositoryImpl[entity.TransferRecord]
}

func NewTransferRecordRepository(db *gorm.DB) TransferRecordRepository {
	return &TransferRecordRepositoryImpl{
		BaseRepositoryImpl: BaseRepositoryImpl[entity.TransferRecord]{db: db},
	}
}

func (r *TransferRecordRepositoryImpl) CreateRecord(record *entity.TransferRecord) error {
	if record == nil {
		return gorm.ErrInvalidData
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if record.ResourceID != nil {
			var previous entity.TransferRecord
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("resource_id = ?", *record.ResourceID).
				Order("id DESC").First(&previous).Error
			if err == nil {
				record.ParentID = &previous.ID
				record.TraceID = previous.TraceID
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if record.TraceID == "" {
				record.TraceID = fmt.Sprintf("resource-%d", *record.ResourceID)
			}
		}
		if record.TraceID == "" {
			record.TraceID = fmt.Sprintf("operation-%d", time.Now().UnixNano())
		}
		if record.OccurredAt.IsZero() {
			record.OccurredAt = time.Now()
		}
		if record.Status == "" {
			record.Status = entity.TransferRecordStatusSucceeded
		}
		if record.CleanupStatus == "" {
			record.CleanupStatus = entity.TransferCleanupPending
		}
		if strings.TrimSpace(record.Metadata) == "" {
			record.Metadata = "{}"
		}
		return tx.Create(record).Error
	})
}

func (r *TransferRecordRepositoryImpl) FindWithFilters(filter TransferRecordFilter) ([]entity.TransferRecord, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	query := r.db.Model(&entity.TransferRecord{})
	if value := strings.TrimSpace(filter.Query); value != "" {
		like := "%" + value + "%"
		query = query.Where(
			"resource_title ILIKE ? OR source_url ILIKE ? OR result_url ILIKE ? OR account_username ILIKE ? OR account_remark ILIKE ? OR trace_id ILIKE ? OR file_id ILIKE ?",
			like, like, like, like, like, like, like,
		)
	}
	if filter.Operation != "" {
		query = query.Where("operation = ?", filter.Operation)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CleanupStatus != "" {
		query = query.Where("cleanup_status = ?", filter.CleanupStatus)
	}
	if filter.PanType != "" {
		query = query.Where("pan_type = ?", filter.PanType)
	}
	if filter.StartAt != nil {
		query = query.Where("occurred_at >= ?", *filter.StartAt)
	}
	if filter.EndAt != nil {
		query = query.Where("occurred_at <= ?", *filter.EndAt)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var records []entity.TransferRecord
	err := query.Order("occurred_at DESC, id DESC").
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Find(&records).Error
	return records, total, err
}

func (r *TransferRecordRepositoryImpl) GetSummary(now time.Time) (*entity.TransferRecordSummary, error) {
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var summary entity.TransferRecordSummary
	err := r.db.Model(&entity.TransferRecord{}).Select(`
		COUNT(*) AS total_records,
		COALESCE(SUM(CASE WHEN occurred_at >= ? THEN 1 ELSE 0 END), 0) AS today_records,
		COALESCE(SUM(CASE WHEN operation = ? THEN 1 ELSE 0 END), 0) AS transfer_count,
		COALESCE(SUM(CASE WHEN operation = ? THEN 1 ELSE 0 END), 0) AS share_count,
		COALESCE(SUM(CASE WHEN cleanup_status = ? THEN 1 ELSE 0 END), 0) AS pending_cleanup,
		COALESCE(SUM(CASE WHEN cleanup_status = ? THEN 1 ELSE 0 END), 0) AS cleaned_count,
		COALESCE(SUM(CASE WHEN cleanup_status = ? THEN 1 ELSE 0 END), 0) AS cleanup_failed`,
		startOfDay,
		entity.TransferOperationTransfer,
		entity.TransferOperationShare,
		entity.TransferCleanupPending,
		entity.TransferCleanupCleaned,
		entity.TransferCleanupFailed,
	).Scan(&summary).Error
	return &summary, err
}

func (r *TransferRecordRepositoryImpl) FindDueForCleanup(now time.Time, limit int) ([]entity.TransferRecord, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var records []entity.TransferRecord
	err := r.db.Where("cleanup_due_at IS NOT NULL AND cleanup_due_at <= ?", now).
		Where("cleanup_status = ? OR (cleanup_status = ? AND cleanup_attempts < ?)",
			entity.TransferCleanupPending, entity.TransferCleanupFailed, maxTransferCleanupAttempts).
		Order("cleanup_due_at ASC, id ASC").Limit(limit).Find(&records).Error
	return records, err
}

func (r *TransferRecordRepositoryImpl) ExtendPendingCleanup(resourceID uint, accountID uint, fileID string, dueAt time.Time) error {
	return r.db.Model(&entity.TransferRecord{}).
		Where("resource_id = ? AND account_id = ? AND file_id = ?", resourceID, accountID, fileID).
		Where("cleanup_status IN ?", []string{entity.TransferCleanupPending, entity.TransferCleanupFailed}).
		UpdateColumns(map[string]interface{}{
			"cleanup_due_at":          dueAt,
			"cleanup_status":          entity.TransferCleanupPending,
			"cleanup_attempts":        0,
			"cleanup_error":           "",
			"last_cleanup_attempt_at": nil,
			"updated_at":              time.Now(),
		}).Error
}

func (r *TransferRecordRepositoryImpl) IsFileDueForCleanup(accountID uint, fileID string, now time.Time) (bool, error) {
	if accountID == 0 || strings.TrimSpace(fileID) == "" {
		return false, nil
	}
	var laterRecords int64
	err := r.db.Model(&entity.TransferRecord{}).
		Where("account_id = ? AND file_id = ?", accountID, fileID).
		Where("cleanup_status IN ?", []string{entity.TransferCleanupPending, entity.TransferCleanupFailed}).
		Where("cleanup_due_at IS NULL OR cleanup_due_at > ?", now).
		Count(&laterRecords).Error
	return laterRecords == 0, err
}

func (r *TransferRecordRepositoryImpl) MarkFileCleaned(accountID uint, fileID string, cleanedAt time.Time) error {
	return r.db.Model(&entity.TransferRecord{}).
		Where("account_id = ? AND file_id = ?", accountID, fileID).
		Where("cleanup_status IN ?", []string{entity.TransferCleanupPending, entity.TransferCleanupFailed}).
		UpdateColumns(map[string]interface{}{
			"cleanup_status":          entity.TransferCleanupCleaned,
			"cleaned_at":              cleanedAt,
			"cleanup_error":           "",
			"cleanup_attempts":        gorm.Expr("cleanup_attempts + 1"),
			"last_cleanup_attempt_at": cleanedAt,
			"updated_at":              cleanedAt,
		}).Error
}

func (r *TransferRecordRepositoryImpl) MarkFileCleanupError(accountID uint, fileID string, message string, attemptedAt time.Time) error {
	return r.db.Model(&entity.TransferRecord{}).
		Where("account_id = ? AND file_id = ?", accountID, fileID).
		Where("cleanup_status IN ?", []string{entity.TransferCleanupPending, entity.TransferCleanupFailed}).
		UpdateColumns(map[string]interface{}{
			"cleanup_status":          entity.TransferCleanupFailed,
			"cleanup_error":           message,
			"cleanup_attempts":        gorm.Expr("cleanup_attempts + 1"),
			"last_cleanup_attempt_at": attemptedAt,
			"updated_at":              attemptedAt,
		}).Error
}

func (r *TransferRecordRepositoryImpl) MarkCleanupError(id uint, message string, attemptedAt time.Time) error {
	return r.db.Model(&entity.TransferRecord{}).Where("id = ?", id).UpdateColumns(map[string]interface{}{
		"cleanup_status":          entity.TransferCleanupFailed,
		"cleanup_error":           message,
		"cleanup_attempts":        gorm.Expr("cleanup_attempts + 1"),
		"last_cleanup_attempt_at": attemptedAt,
		"updated_at":              attemptedAt,
	}).Error
}

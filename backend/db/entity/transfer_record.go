package entity

import "time"

const (
	TransferOperationTransfer = "transfer"
	TransferOperationShare    = "share"

	TransferRecordStatusSucceeded = "succeeded"
	TransferRecordStatusFailed    = "failed"

	TransferCleanupPending     = "pending"
	TransferCleanupCleaned     = "cleaned"
	TransferCleanupFailed      = "failed"
	TransferCleanupNotRequired = "not_required"
)

// TransferRecord permanently snapshots a transfer/share operation. Credentials
// are intentionally excluded; account fields are display-only historical data.
type TransferRecord struct {
	ID                   uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	TraceID              string     `json:"trace_id" gorm:"size:96;index;not null"`
	ParentID             *uint      `json:"parent_id" gorm:"index"`
	ResourceID           *uint      `json:"resource_id" gorm:"index"`
	ResourceKey          string     `json:"resource_key" gorm:"size:64;index"`
	ResourceSource       string     `json:"resource_source" gorm:"size:32;index"`
	ExternalID           string     `json:"external_id" gorm:"size:128;index"`
	TaskID               *uint      `json:"task_id" gorm:"index"`
	TaskItemID           *uint      `json:"task_item_id" gorm:"index"`
	Operation            string     `json:"operation" gorm:"size:20;index;not null"`
	TriggerSource        string     `json:"trigger_source" gorm:"size:32;index"`
	Status               string     `json:"status" gorm:"size:20;index;not null"`
	SourceURL            string     `json:"source_url" gorm:"size:2048"`
	PreviousShareURL     string     `json:"previous_share_url" gorm:"size:2048"`
	ResultURL            string     `json:"result_url" gorm:"size:2048"`
	ResourceTitle        string     `json:"resource_title" gorm:"size:255;index"`
	PanID                *uint      `json:"pan_id" gorm:"index"`
	PanType              string     `json:"pan_type" gorm:"size:32;index"`
	PanName              string     `json:"pan_name" gorm:"size:64"`
	AccountID            *uint      `json:"account_id" gorm:"index"`
	AccountUsername      string     `json:"account_username" gorm:"size:100"`
	AccountRemark        string     `json:"account_remark" gorm:"size:100"`
	FileID               string     `json:"file_id" gorm:"size:512;index"`
	OccurredAt           time.Time  `json:"occurred_at" gorm:"index;not null"`
	CleanupDueAt         *time.Time `json:"cleanup_due_at" gorm:"index"`
	CleanupStatus        string     `json:"cleanup_status" gorm:"size:20;index"`
	CleanupAttempts      int        `json:"cleanup_attempts" gorm:"default:0"`
	LastCleanupAttemptAt *time.Time `json:"last_cleanup_attempt_at"`
	CleanedAt            *time.Time `json:"cleaned_at" gorm:"index"`
	CleanupError         string     `json:"cleanup_error" gorm:"type:text"`
	ErrorMessage         string     `json:"error_message" gorm:"type:text"`
	DurationMS           int64      `json:"duration_ms"`
	Metadata             string     `json:"metadata" gorm:"type:jsonb;default:'{}'"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (TransferRecord) TableName() string {
	return "transfer_records"
}

type TransferRecordSummary struct {
	TotalRecords   int64 `json:"total_records"`
	TodayRecords   int64 `json:"today_records"`
	TransferCount  int64 `json:"transfer_count"`
	ShareCount     int64 `json:"share_count"`
	PendingCleanup int64 `json:"pending_cleanup"`
	CleanedCount   int64 `json:"cleaned_count"`
	CleanupFailed  int64 `json:"cleanup_failed"`
}

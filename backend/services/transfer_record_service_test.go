package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
)

type transferRecordRepoFake struct {
	repo.TransferRecordRepository
	created          *entity.TransferRecord
	extendedResource uint
	extendedAccount  uint
	extendedFile     string
	extendedDue      time.Time
}

func (f *transferRecordRepoFake) CreateRecord(record *entity.TransferRecord) error {
	copy := *record
	f.created = &copy
	return nil
}

func (f *transferRecordRepoFake) ExtendPendingCleanup(resourceID, accountID uint, fileID string, dueAt time.Time) error {
	f.extendedResource = resourceID
	f.extendedAccount = accountID
	f.extendedFile = fileID
	f.extendedDue = dueAt
	return nil
}

type transferPanRepoFake struct {
	repo.PanRepository
	pan *entity.Pan
}

func (f *transferPanRepoFake) FindByID(uint) (*entity.Pan, error) { return f.pan, nil }

func TestTransferRecordServiceRecordSuccess(t *testing.T) {
	recordRepo := &transferRecordRepoFake{}
	service := NewTransferRecordService(recordRepo, &transferPanRepoFake{pan: &entity.Pan{
		ID: 1, Name: "quark", Remark: "夸克网盘",
	}})
	occurredAt := time.Date(2026, 9, 1, 10, 30, 0, 0, time.Local)
	panID := uint(1)
	resource := &entity.Resource{
		ID: 23, Title: "测试资源", URL: "https://pan.quark.cn/s/source", SaveURL: "https://pan.quark.cn/s/result",
		Fid: "file-23", Key: "abc123", Source: "telegram", ExternalID: "external-23", PanID: &panID,
	}
	account := &entity.Cks{
		ID: 7, PanID: 1, Username: "user@example.com", Remark: "中转一号", ServiceType: "quark", Ck: "secret-cookie",
	}

	record, err := service.RecordSuccess(resource, account, TransferRecordInput{
		Operation:     entity.TransferOperationTransfer,
		TriggerSource: "admin_transfer_task",
		OccurredAt:    occurredAt,
		Metadata:      map[string]interface{}{"batch": "manual"},
	})
	if err != nil {
		t.Fatalf("RecordSuccess() error = %v", err)
	}
	if recordRepo.created == nil || recordRepo.created != nil && recordRepo.created.ResourceID == nil {
		t.Fatal("expected an audit record with resource id")
	}
	if record.ResourceTitle != resource.Title || record.ResourceSource != resource.Source || record.ExternalID != resource.ExternalID {
		t.Fatalf("resource snapshot mismatch: %+v", record)
	}
	if record.AccountUsername != account.Username || record.AccountRemark != account.Remark || record.PanType != "quark" || record.PanName != "夸克网盘" {
		t.Fatalf("account/platform snapshot mismatch: %+v", record)
	}
	wantDue := occurredAt.Add(10 * time.Minute)
	if record.CleanupDueAt == nil || !record.CleanupDueAt.Equal(wantDue) {
		t.Fatalf("cleanup due = %v, want %v", record.CleanupDueAt, wantDue)
	}
	if recordRepo.extendedResource != resource.ID || recordRepo.extendedAccount != account.ID || recordRepo.extendedFile != resource.Fid || !recordRepo.extendedDue.Equal(wantDue) {
		t.Fatalf("pending cleanup extension mismatch: %+v", recordRepo)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal audit record: %v", err)
	}
	if strings.Contains(string(raw), account.Ck) {
		t.Fatal("audit record must not contain account credentials")
	}
}

func TestTransferRecordServiceRecordFailure(t *testing.T) {
	recordRepo := &transferRecordRepoFake{}
	service := NewTransferRecordService(recordRepo, &transferPanRepoFake{pan: &entity.Pan{
		ID: 1, Name: "quark", Remark: "夸克网盘",
	}})
	panID := uint(1)
	resource := &entity.Resource{
		ID: 31, Title: "失败资源", URL: "https://pan.quark.cn/s/source", Source: "quanpan", PanID: &panID,
	}
	account := &entity.Cks{
		ID: 9, PanID: 1, Username: "transfer@example.com", Remark: "中转二号", ServiceType: "quark",
	}

	record, err := service.RecordFailure(resource, account, TransferRecordInput{
		TriggerSource: "resource_link",
		ErrorMessage:  "容量不足",
		DurationMS:    1520,
	})
	if err != nil {
		t.Fatalf("RecordFailure() error = %v", err)
	}
	if record.Status != entity.TransferRecordStatusFailed || record.ErrorMessage != "容量不足" {
		t.Fatalf("failure state mismatch: %+v", record)
	}
	if record.CleanupStatus != entity.TransferCleanupNotRequired || record.CleanupDueAt != nil {
		t.Fatalf("failed transfer must not enter cleanup queue: %+v", record)
	}
	if recordRepo.extendedResource != 0 || recordRepo.extendedAccount != 0 || recordRepo.extendedFile != "" {
		t.Fatalf("failed transfer unexpectedly extended cleanup: %+v", recordRepo)
	}
	if record.AccountID == nil || *record.AccountID != account.ID || record.PanName != "夸克网盘" {
		t.Fatalf("account snapshot mismatch: %+v", record)
	}
}

func TestTransferRecordServiceRecordFailureWithoutPersistedResource(t *testing.T) {
	recordRepo := &transferRecordRepoFake{}
	service := NewTransferRecordService(recordRepo, nil)
	taskID := uint(11)
	taskItemID := uint(12)

	record, err := service.RecordFailure(nil, nil, TransferRecordInput{
		TriggerSource:  "admin_transfer_task",
		ResourceTitle:  "待转存资源",
		ResourceSource: "task",
		SourceURL:      "https://pan.xunlei.com/s/source",
		PanType:        "xunlei",
		ErrorMessage:   "分享状态异常",
		TaskID:         &taskID,
		TaskItemID:     &taskItemID,
	})
	if err != nil {
		t.Fatalf("RecordFailure() error = %v", err)
	}
	if record.ResourceID != nil || record.ResourceTitle != "待转存资源" || record.SourceURL == "" {
		t.Fatalf("unpersisted resource snapshot mismatch: %+v", record)
	}
	if record.TaskID == nil || *record.TaskID != taskID || record.TaskItemID == nil || *record.TaskItemID != taskItemID {
		t.Fatalf("task snapshot mismatch: %+v", record)
	}
	if record.PanType != "xunlei" || record.PanName != "xunlei" {
		t.Fatalf("platform snapshot mismatch: %+v", record)
	}
}

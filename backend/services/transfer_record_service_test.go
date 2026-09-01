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

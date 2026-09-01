package services

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
)

const TransferFileRetention = 10 * time.Minute

var errTransferRecordServiceUnavailable = errors.New("transfer record service is unavailable")

type TransferRecordInput struct {
	Operation        string
	TriggerSource    string
	PreviousShareURL string
	ResultURL        string
	FileID           string
	OccurredAt       time.Time
	TaskID           *uint
	TaskItemID       *uint
	DurationMS       int64
	Metadata         map[string]interface{}
}

type TransferRecordService struct {
	recordRepo repo.TransferRecordRepository
	panRepo    repo.PanRepository
}

func NewTransferRecordService(recordRepo repo.TransferRecordRepository, panRepo repo.PanRepository) *TransferRecordService {
	return &TransferRecordService{recordRepo: recordRepo, panRepo: panRepo}
}

func (s *TransferRecordService) RecordSuccess(resource *entity.Resource, account *entity.Cks, input TransferRecordInput) (*entity.TransferRecord, error) {
	if s == nil || s.recordRepo == nil {
		return nil, errTransferRecordServiceUnavailable
	}
	if resource == nil {
		return nil, errors.New("resource is required")
	}

	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	operation := strings.TrimSpace(input.Operation)
	if operation == "" {
		operation = entity.TransferOperationTransfer
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		fileID = strings.TrimSpace(resource.Fid)
	}
	resultURL := strings.TrimSpace(input.ResultURL)
	if resultURL == "" {
		resultURL = strings.TrimSpace(resource.SaveURL)
	}

	var accountID *uint
	var accountUsername, accountRemark, panType string
	var panID *uint
	if resource.PanID != nil {
		value := *resource.PanID
		panID = &value
	}
	if account != nil {
		value := account.ID
		accountID = &value
		accountUsername = account.Username
		accountRemark = account.Remark
		panType = account.ServiceType
		if account.PanID != 0 {
			value := account.PanID
			panID = &value
		}
	}

	var panName string
	if panID != nil && s.panRepo != nil {
		if panInfo, err := s.panRepo.FindByID(*panID); err == nil && panInfo != nil {
			if panType == "" {
				panType = panInfo.Name
			}
			panName = panInfo.Remark
			if panName == "" {
				panName = panInfo.Name
			}
		}
	}
	if panName == "" {
		panName = panType
	}

	metadata := "{}"
	if len(input.Metadata) > 0 {
		if raw, err := json.Marshal(input.Metadata); err == nil {
			metadata = string(raw)
		}
	}
	dueAt := occurredAt.Add(TransferFileRetention)
	record := &entity.TransferRecord{
		ResourceID:       &resource.ID,
		ResourceKey:      resource.Key,
		ResourceSource:   resource.Source,
		ExternalID:       resource.ExternalID,
		TaskID:           input.TaskID,
		TaskItemID:       input.TaskItemID,
		Operation:        operation,
		TriggerSource:    strings.TrimSpace(input.TriggerSource),
		Status:           entity.TransferRecordStatusSucceeded,
		SourceURL:        resource.URL,
		PreviousShareURL: input.PreviousShareURL,
		ResultURL:        resultURL,
		ResourceTitle:    resource.Title,
		PanID:            panID,
		PanType:          panType,
		PanName:          panName,
		AccountID:        accountID,
		AccountUsername:  accountUsername,
		AccountRemark:    accountRemark,
		FileID:           fileID,
		OccurredAt:       occurredAt,
		CleanupDueAt:     &dueAt,
		CleanupStatus:    entity.TransferCleanupPending,
		DurationMS:       input.DurationMS,
		Metadata:         metadata,
	}

	if accountID != nil && fileID != "" {
		if err := s.recordRepo.ExtendPendingCleanup(resource.ID, *accountID, fileID, dueAt); err != nil {
			return nil, err
		}
	}
	if err := s.recordRepo.CreateRecord(record); err != nil {
		return nil, err
	}
	return record, nil
}

var defaultTransferRecordService struct {
	sync.RWMutex
	service *TransferRecordService
}

func SetDefaultTransferRecordService(service *TransferRecordService) {
	defaultTransferRecordService.Lock()
	defaultTransferRecordService.service = service
	defaultTransferRecordService.Unlock()
}

func RecordSuccessfulTransfer(resource *entity.Resource, account *entity.Cks, input TransferRecordInput) (*entity.TransferRecord, error) {
	defaultTransferRecordService.RLock()
	service := defaultTransferRecordService.service
	defaultTransferRecordService.RUnlock()
	if service == nil {
		return nil, errTransferRecordServiceUnavailable
	}
	return service.RecordSuccess(resource, account, input)
}

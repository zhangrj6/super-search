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
	ResourceKey      string
	ResourceSource   string
	ExternalID       string
	SourceURL        string
	ResourceTitle    string
	PanID            *uint
	PanType          string
	PreviousShareURL string
	ResultURL        string
	FileID           string
	ErrorMessage     string
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
	if resource == nil {
		return nil, errors.New("resource is required")
	}
	return s.record(resource, account, input, entity.TransferRecordStatusSucceeded)
}

func (s *TransferRecordService) RecordFailure(resource *entity.Resource, account *entity.Cks, input TransferRecordInput) (*entity.TransferRecord, error) {
	return s.record(resource, account, input, entity.TransferRecordStatusFailed)
}

func (s *TransferRecordService) record(resource *entity.Resource, account *entity.Cks, input TransferRecordInput, status string) (*entity.TransferRecord, error) {
	if s == nil || s.recordRepo == nil {
		return nil, errTransferRecordServiceUnavailable
	}

	occurredAt := input.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	operation := strings.TrimSpace(input.Operation)
	if operation == "" {
		operation = entity.TransferOperationTransfer
	}

	var resourceID *uint
	resourceKey := strings.TrimSpace(input.ResourceKey)
	resourceSource := strings.TrimSpace(input.ResourceSource)
	externalID := strings.TrimSpace(input.ExternalID)
	sourceURL := strings.TrimSpace(input.SourceURL)
	resourceTitle := strings.TrimSpace(input.ResourceTitle)
	var panID *uint
	if input.PanID != nil {
		value := *input.PanID
		panID = &value
	}
	if resource != nil {
		if resource.ID != 0 {
			value := resource.ID
			resourceID = &value
		}
		resourceKey = resource.Key
		resourceSource = resource.Source
		externalID = resource.ExternalID
		sourceURL = resource.URL
		resourceTitle = resource.Title
		if resource.PanID != nil {
			value := *resource.PanID
			panID = &value
		}
	}

	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" && resource != nil && (status == entity.TransferRecordStatusSucceeded || operation == entity.TransferOperationShare) {
		fileID = strings.TrimSpace(resource.Fid)
	}
	resultURL := strings.TrimSpace(input.ResultURL)
	if resultURL == "" && resource != nil && status == entity.TransferRecordStatusSucceeded {
		resultURL = strings.TrimSpace(resource.SaveURL)
	}

	var accountID *uint
	var accountUsername, accountRemark string
	panType := strings.TrimSpace(input.PanType)
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
	record := &entity.TransferRecord{
		ResourceID:       resourceID,
		ResourceKey:      resourceKey,
		ResourceSource:   resourceSource,
		ExternalID:       externalID,
		TaskID:           input.TaskID,
		TaskItemID:       input.TaskItemID,
		Operation:        operation,
		TriggerSource:    strings.TrimSpace(input.TriggerSource),
		Status:           status,
		SourceURL:        sourceURL,
		PreviousShareURL: input.PreviousShareURL,
		ResultURL:        resultURL,
		ResourceTitle:    resourceTitle,
		PanID:            panID,
		PanType:          panType,
		PanName:          panName,
		AccountID:        accountID,
		AccountUsername:  accountUsername,
		AccountRemark:    accountRemark,
		FileID:           fileID,
		OccurredAt:       occurredAt,
		ErrorMessage:     strings.TrimSpace(input.ErrorMessage),
		DurationMS:       input.DurationMS,
		Metadata:         metadata,
	}

	if status == entity.TransferRecordStatusSucceeded {
		dueAt := occurredAt.Add(TransferFileRetention)
		record.CleanupDueAt = &dueAt
		record.CleanupStatus = entity.TransferCleanupPending
		if resourceID != nil && accountID != nil && fileID != "" {
			if err := s.recordRepo.ExtendPendingCleanup(*resourceID, *accountID, fileID, dueAt); err != nil {
				return nil, err
			}
		}
	} else {
		record.CleanupStatus = entity.TransferCleanupNotRequired
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

func RecordFailedTransfer(resource *entity.Resource, account *entity.Cks, input TransferRecordInput) (*entity.TransferRecord, error) {
	defaultTransferRecordService.RLock()
	service := defaultTransferRecordService.service
	defaultTransferRecordService.RUnlock()
	if service == nil {
		return nil, errTransferRecordServiceUnavailable
	}
	return service.RecordFailure(resource, account, input)
}

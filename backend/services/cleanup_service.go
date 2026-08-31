package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	pan "github.com/ctwj/urldb/common"
	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
	"github.com/ctwj/urldb/utils"
)

// CleanupService 转存文件自动清理服务
// 周期性扫描已转存且超过保留期的资源，调用网盘 API 删除文件并清空转存字段
type CleanupService struct {
	resourceRepo       repo.ResourceRepository
	transferRecordRepo repo.TransferRecordRepository
	configRepo         repo.SystemConfigRepository
	cksRepo            repo.CksRepository
	panRepo            repo.PanRepository
}

// NewCleanupService 创建清理服务
func NewCleanupService(
	resourceRepo repo.ResourceRepository,
	configRepo repo.SystemConfigRepository,
	cksRepo repo.CksRepository,
	panRepo repo.PanRepository,
	transferRecordRepos ...repo.TransferRecordRepository,
) *CleanupService {
	var transferRecordRepo repo.TransferRecordRepository
	if len(transferRecordRepos) > 0 {
		transferRecordRepo = transferRecordRepos[0]
	}
	return &CleanupService{
		resourceRepo:       resourceRepo,
		transferRecordRepo: transferRecordRepo,
		configRepo:         configRepo,
		cksRepo:            cksRepo,
		panRepo:            panRepo,
	}
}

// Run 执行一次清理扫描
// 返回处理统计：总数、成功数、失败数
func (s *CleanupService) Run(ctx context.Context) (total, success, failed int, err error) {
	startTime := time.Now()
	utils.Info("[CleanupService] 开始执行清理任务")
	const batchLimit = 100
	now := time.Now()
	accountCache := make(map[uint]*entity.Cks)
	processedFiles := make(map[string]struct{})

	if s.transferRecordRepo != nil {
		records, findErr := s.transferRecordRepo.FindDueForCleanup(now, batchLimit)
		if findErr != nil {
			utils.Error("[CleanupService] 查询审计清理队列失败: %v", findErr)
			return 0, 0, 0, findErr
		}

		groups := groupCleanupRecords(records)
		for _, group := range groups {
			select {
			case <-ctx.Done():
				utils.Warn("[CleanupService] 任务被取消，已处理 %d 条文件", success+failed)
				return total, success, failed, nil
			default:
			}

			record := group[0]
			if record.AccountID == nil || *record.AccountID == 0 || strings.TrimSpace(record.FileID) == "" {
				total++
				message := "审计记录缺少 account_id 或 file_id"
				for _, item := range group {
					_ = s.transferRecordRepo.MarkCleanupError(item.ID, message, now)
				}
				failed++
				continue
			}

			accountID := *record.AccountID
			fileKey := cleanupFileKey(accountID, record.FileID)
			processedFiles[fileKey] = struct{}{}
			due, dueErr := s.transferRecordRepo.IsFileDueForCleanup(accountID, record.FileID, now)
			if dueErr != nil {
				total++
				_ = s.transferRecordRepo.MarkFileCleanupError(accountID, record.FileID, truncateMsg(dueErr.Error()), now)
				failed++
				continue
			}
			if !due {
				continue
			}

			total++
			account, accErr := s.resolveAccountByID(accountID, accountCache)
			if accErr != nil {
				utils.Error("[CleanupService] 审计文件 %s 解析账号失败: %v", record.FileID, accErr)
				_ = s.transferRecordRepo.MarkFileCleanupError(accountID, record.FileID, truncateMsg(accErr.Error()), now)
				failed++
				continue
			}

			delErr := s.deleteFile(account, record.FileID)
			if delErr == nil || isFileNotExist(delErr) {
				cleanedAt := time.Now()
				if markErr := s.transferRecordRepo.MarkFileCleaned(accountID, record.FileID, cleanedAt); markErr != nil {
					utils.Error("[CleanupService] 文件已删除但审计状态更新失败: %v", markErr)
				}
				for _, resourceID := range cleanupResourceIDs(group) {
					if markErr := s.resourceRepo.MarkCleanedIfFileMatches(resourceID, record.FileID, cleanedAt); markErr != nil {
						utils.Error("[CleanupService] 资源 ID=%d 清空转存字段失败: %v", resourceID, markErr)
					}
				}
				success++
				continue
			}

			utils.Error("[CleanupService] 审计文件清理失败 account=%d file=%s: %v", accountID, record.FileID, delErr)
			_ = s.transferRecordRepo.MarkFileCleanupError(accountID, record.FileID, truncateMsg(delErr.Error()), time.Now())
			failed++
		}
	}

	// Legacy resources created before transfer_records existed remain eligible.
	resources, legacyErr := s.resourceRepo.FindDueForCleanup(int(TransferFileRetention/time.Minute), batchLimit)
	if legacyErr != nil {
		utils.Error("[CleanupService] 查询历史清理队列失败: %v", legacyErr)
		return total, success, failed, legacyErr
	}

	for _, res := range resources {
		select {
		case <-ctx.Done():
			utils.Warn("[CleanupService] 任务被取消，已处理 %d 条文件", success+failed)
			return total, success, failed, nil
		default:
		}

		if res.CkID != nil {
			if _, seen := processedFiles[cleanupFileKey(*res.CkID, res.Fid)]; seen {
				continue
			}
		}
		total++

		if res.Fid == "" {
			utils.Warn("[CleanupService] 资源 ID=%d 无 fid，跳过删除并标记失败", res.ID)
			_ = s.resourceRepo.MarkCleanError(res.ID, "fid 为空", time.Now())
			failed++
			continue
		}

		account, accErr := s.resolveAccount(res, accountCache)
		if accErr != nil {
			utils.Error("[CleanupService] 资源 ID=%d 解析账号失败: %v", res.ID, accErr)
			_ = s.resourceRepo.MarkCleanError(res.ID, truncateMsg(accErr.Error()), time.Now())
			failed++
			continue
		}

		delErr := s.deleteFile(account, res.Fid)
		if delErr == nil {
			if err := s.resourceRepo.MarkCleanedIfFileMatches(res.ID, res.Fid, time.Now()); err != nil {
				utils.Error("[CleanupService] 资源 ID=%d 文件已删除但更新数据库失败: %v", res.ID, err)
			} else {
				utils.Info("[CleanupService] 资源 ID=%d 清理成功", res.ID)
			}
			success++
		} else if isFileNotExist(delErr) {
			utils.Info("[CleanupService] 资源 ID=%d 网盘文件已不存在，视为清理成功: %v", res.ID, delErr)
			_ = s.resourceRepo.MarkCleanedIfFileMatches(res.ID, res.Fid, time.Now())
			success++
		} else {
			utils.Error("[CleanupService] 资源 ID=%d 清理失败: %v", res.ID, delErr)
			_ = s.resourceRepo.MarkCleanError(res.ID, truncateMsg(delErr.Error()), time.Now())
			failed++
		}
	}

	elapsed := time.Since(startTime)
	utils.Info("[CleanupService] 清理任务完成，总计=%d, 成功=%d, 失败=%d, 耗时=%v", total, success, failed, elapsed)
	return total, success, failed, nil
}

func groupCleanupRecords(records []entity.TransferRecord) [][]entity.TransferRecord {
	groupsByKey := make(map[string][]entity.TransferRecord)
	order := make([]string, 0)
	for _, record := range records {
		key := fmt.Sprintf("record:%d", record.ID)
		if record.AccountID != nil && *record.AccountID != 0 && strings.TrimSpace(record.FileID) != "" {
			key = cleanupFileKey(*record.AccountID, record.FileID)
		}
		if _, exists := groupsByKey[key]; !exists {
			order = append(order, key)
		}
		groupsByKey[key] = append(groupsByKey[key], record)
	}
	groups := make([][]entity.TransferRecord, 0, len(order))
	for _, key := range order {
		groups = append(groups, groupsByKey[key])
	}
	return groups
}

func cleanupFileKey(accountID uint, fileID string) string {
	return fmt.Sprintf("%d:%s", accountID, fileID)
}

func cleanupResourceIDs(records []entity.TransferRecord) []uint {
	seen := make(map[uint]struct{})
	ids := make([]uint, 0)
	for _, record := range records {
		if record.ResourceID == nil || *record.ResourceID == 0 {
			continue
		}
		if _, exists := seen[*record.ResourceID]; exists {
			continue
		}
		seen[*record.ResourceID] = struct{}{}
		ids = append(ids, *record.ResourceID)
	}
	return ids
}

// resolveAccount 解析资源对应的账号 cookie
// 通过 ck_id 查询账号，确保使用与转存时同一账号进行删除（防跨账号误删）
func (s *CleanupService) resolveAccount(res *entity.Resource, cache map[uint]*entity.Cks) (*entity.Cks, error) {
	if res.CkID == nil {
		return nil, errNoAccountBound
	}
	return s.resolveAccountByID(*res.CkID, cache)
}

func (s *CleanupService) resolveAccountByID(accID uint, cache map[uint]*entity.Cks) (*entity.Cks, error) {

	// 命中缓存
	if acc, ok := cache[accID]; ok && acc != nil {
		return acc, nil
	}

	acc, err := s.cksRepo.FindByIds([]uint{accID})
	if err != nil {
		return nil, err
	}
	if len(acc) == 0 {
		return nil, errNoAccountBound
	}

	cache[accID] = acc[0]
	return acc[0], nil
}

// deleteFile 调用网盘 API 删除指定文件
// 根据账号 ServiceType 选择对应网盘服务实例
func (s *CleanupService) deleteFile(account *entity.Cks, fid string) error {
	serviceType := account.ServiceType
	if serviceType == "" {
		return errUnknownServiceType
	}

	factory := pan.NewPanFactory()
	// 创建带账号 cookie 的配置
	cfg := &pan.PanConfig{
		Cookie: account.Ck,
	}

	// 使用工厂创建对应类型的网盘服务
	service, err := factory.CreatePanServiceByType(toPanServiceType(serviceType), cfg)
	if err != nil {
		return err
	}

	// 设置 cks 仓库（部分操作需要刷新 token）
	if setter, ok := service.(interface {
		SetCKSRepository(repo.CksRepository, entity.Cks)
	}); ok {
		setter.SetCKSRepository(s.cksRepo, *account)
	}

	result, err := service.DeleteFiles([]string{fid})
	if err != nil {
		return err
	}
	if result == nil {
		return errDeleteFailed
	}
	if !result.Success {
		// 将 Message 作为错误返回，便于上层宽松匹配"文件不存在"
		return errMsg(result.Message)
	}
	return nil
}

// toPanServiceType 将 ServiceType 字符串转为枚举
func toPanServiceType(serviceType string) pan.ServiceType {
	switch strings.ToLower(serviceType) {
	case "quark":
		return pan.Quark
	case "alipan", "aliyun":
		return pan.Alipan
	case "baidu":
		return pan.BaiduPan
	case "uc":
		return pan.UC
	case "xunlei":
		return pan.Xunlei
	default:
		return pan.NotFound
	}
}

// isFileNotExist 判断错误是否表示"文件已不存在"
// 宽松匹配中英文关键字，避免被具体错误码绑死（FR-009）
func isFileNotExist(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "不存在") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such") ||
		strings.Contains(msg, "already deleted") ||
		strings.Contains(msg, "已删除")
}

// truncateMsg 截断错误信息以匹配字段长度（255）
func truncateMsg(msg string) string {
	if len(msg) > 255 {
		return msg[:255]
	}
	return msg
}

// 错误变量
var (
	errNoAccountBound     = errMsg("资源未绑定账号，无法确定清理使用的 cookie")
	errUnknownServiceType = errMsg("账号 ServiceType 为空，无法确定网盘类型")
	errDeleteFailed       = errMsg("删除文件失败：返回结果为空")
)

// errMsg 简单包装字符串为 error，便于上层匹配关键字
type errMsg string

func (e errMsg) Error() string { return string(e) }

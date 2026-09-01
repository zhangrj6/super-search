package services

import (
	"fmt"
	"strings"
	"time"

	panutils "github.com/ctwj/urldb/common"
	"github.com/ctwj/urldb/db/entity"
	"github.com/ctwj/urldb/db/repo"
	"github.com/ctwj/urldb/utils"
)

// PerformShare 对系统已存文件按 fid 重新生成分享链接（saveUrl 失效时的恢复操作，FR-015）。
// 镜像 PerformAutoTransfer 的结构：按 resource.CkID 取持有账号 → 创建网盘服务 → 类型断言 Sharer → Share(fid)。
// 刷新 save_url 后会重置 10 分钟清理计时，并追加一条永久分享审计记录。
// 失败时调用方（决策树 ResolveWithCheck）会回退到「判原始→转存」，故此处失败是安全的降级。
func PerformShare(cksRepo repo.CksRepository, resourceRepo repo.ResourceRepository, resource *entity.Resource, triggerSources ...string) TransferResult {
	startedAt := time.Now()
	if resource == nil {
		return TransferResult{Success: false, ErrorMsg: "resource 为空"}
	}
	if resource.CkID == nil || resource.Fid == "" {
		return TransferResult{Success: false, ErrorMsg: "资源无已转存文件信息（缺少 ck_id/fid）"}
	}

	account, err := cksRepo.FindByID(*resource.CkID)
	if err != nil || account == nil {
		utils.Error("[SHARE] 取持有账号失败 (ck_id=%d): %v", *resource.CkID, err)
		return TransferResult{Success: false, ErrorMsg: fmt.Sprintf("取持有账号失败: %v", err)}
	}

	factory := panutils.NewPanFactory()
	service, err := factory.CreatePanService(resource.URL, &panutils.PanConfig{
		URL:    resource.URL,
		Cookie: account.Ck,
	})
	if err != nil {
		utils.Error("[SHARE] 创建网盘服务失败: %v", err)
		return TransferResult{Success: false, ErrorMsg: fmt.Sprintf("创建网盘服务失败: %v", err)}
	}
	service.SetCKSRepository(cksRepo, *account)

	sharer, ok := service.(panutils.Sharer)
	if !ok {
		return TransferResult{Success: false, ErrorMsg: "该网盘平台不支持重新分享"}
	}

	result, err := sharer.Share(resource.Fid)
	if err != nil {
		utils.Error("[SHARE] 重新分享失败 (resource=%d): %v", resource.ID, err)
		return TransferResult{Success: false, ErrorMsg: fmt.Sprintf("重新分享失败: %v", err)}
	}
	if result == nil || !result.Success || result.ShareURL == "" {
		msg := "重新分享失败"
		if result != nil && result.Message != "" {
			msg = result.Message
		}
		utils.Warn("[SHARE] 重新分享未成功 (resource=%d): %s", resource.ID, msg)
		return TransferResult{Success: false, ErrorMsg: msg}
	}

	previousShareURL := resource.SaveURL
	now := time.Now()
	resource.SaveURL = result.ShareURL
	resource.TransferredAt = &now
	if err := resourceRepo.UpdateFields(resource.ID, map[string]interface{}{
		"save_url":            result.ShareURL,
		"transferred_at":      now,
		"updated_at":          now,
		"cleaned_at":          nil,
		"clean_error_msg":     "",
		"last_clean_error_at": nil,
	}); err != nil {
		utils.Error("[SHARE] 更新 save_url 失败: %v", err)
		return TransferResult{Success: false, ErrorMsg: fmt.Sprintf("更新 save_url 失败: %v", err)}
	}

	triggerSource := "reshare"
	if len(triggerSources) > 0 && strings.TrimSpace(triggerSources[0]) != "" {
		triggerSource = strings.TrimSpace(triggerSources[0])
	}
	if _, err := RecordSuccessfulTransfer(resource, account, TransferRecordInput{
		Operation:        entity.TransferOperationShare,
		TriggerSource:    triggerSource,
		PreviousShareURL: previousShareURL,
		ResultURL:        result.ShareURL,
		FileID:           resource.Fid,
		OccurredAt:       now,
		DurationMS:       time.Since(startedAt).Milliseconds(),
	}); err != nil {
		utils.Error("[SHARE] 记录分享链路失败 (resource=%d): %v", resource.ID, err)
	}

	utils.Info("[SHARE] 重新分享成功 - resource=%d, save_url=%s", resource.ID, result.ShareURL)
	return TransferResult{Success: true, SaveURL: result.ShareURL, Fid: resource.Fid}
}

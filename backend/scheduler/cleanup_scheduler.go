package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/ctwj/urldb/services"
	"github.com/ctwj/urldb/utils"
)

// CleanupScheduler 转存文件自动清理调度器
// 周期性扫描已转存且超过保留期的资源，调用 CleanupService 执行清理
type CleanupScheduler struct {
	*BaseScheduler
	cleanupService  *services.CleanupService
	cleanupRunning  bool
	processingMutex sync.Mutex // 防止清理任务重叠执行
}

// NewCleanupScheduler 创建自动清理调度器
func NewCleanupScheduler(base *BaseScheduler, cleanupService *services.CleanupService) *CleanupScheduler {
	return &CleanupScheduler{
		BaseScheduler:   base,
		cleanupService:  cleanupService,
		cleanupRunning:  false,
		processingMutex: sync.Mutex{},
	}
}

// Start 启动自动清理定时任务
func (c *CleanupScheduler) Start() {
	if c.cleanupRunning {
		utils.Debug("转存文件自动清理任务已在运行中")
		return
	}

	c.cleanupRunning = true
	utils.Info("启动转存文件自动清理定时任务")

	go func() {
		interval := time.Minute
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		utils.Info("转存文件自动清理任务已启动，调度周期: %v", interval)
		c.triggerRun()

		for {
			select {
			case <-ticker.C:
				c.triggerRun()
			case <-c.GetStopChan():
				utils.Info("停止转存文件自动清理定时任务")
				return
			}
		}
	}()
}

func (c *CleanupScheduler) triggerRun() {
	if c.processingMutex.TryLock() {
		go func() {
			defer c.processingMutex.Unlock()
			c.runOnce(context.Background())
		}()
		return
	}
	utils.Debug("上一次自动清理任务还在执行中，跳过本次执行")
}

// Stop 停止自动清理定时任务
func (c *CleanupScheduler) Stop() {
	if !c.cleanupRunning {
		utils.Debug("转存文件自动清理任务未在运行")
		return
	}

	c.GetStopChan() <- true
	c.cleanupRunning = false
	utils.Info("已发送停止信号给转存文件自动清理任务")
}

// IsCleanupRunning 检查自动清理任务是否在运行
func (c *CleanupScheduler) IsCleanupRunning() bool {
	return c.cleanupRunning
}

// runOnce executes the mandatory fixed-retention cleanup policy.
func (c *CleanupScheduler) runOnce(ctx context.Context) {
	total, success, failed, runErr := c.cleanupService.Run(ctx)
	if runErr != nil {
		utils.Error("[CleanupScheduler] 清理任务执行异常: %v", runErr)
		return
	}
	utils.Info("[CleanupScheduler] 本轮清理结束: 总计=%d, 成功=%d, 失败=%d", total, success, failed)
}

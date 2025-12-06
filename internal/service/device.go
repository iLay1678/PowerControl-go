package service

import (
	"context"
	"fmt"
	"powercontrol/internal/bemfa"
	"powercontrol/internal/config"
	"powercontrol/internal/core"
	"powercontrol/internal/logger"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// DeviceService 设备服务
type DeviceService struct {
	device      *config.Device
	cfg         *config.Config
	log         *zap.SugaredLogger
	bemfaClient *bemfa.Client
	cron        *cron.Cron
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	status      string // "online", "offline", "unknown"
	mu          sync.RWMutex
}

// NewDeviceService 创建设备服务
func NewDeviceService(device *config.Device, cfg *config.Config) *DeviceService {
	return &DeviceService{
		device: device,
		cfg:    cfg,
		log:    logger.GetLogger(fmt.Sprintf("device_%s", device.Name)),
		cron:   cron.New(),
		status: "unknown",
	}
}

// Start 启动设备服务
func (s *DeviceService) Start(ctx context.Context) error {
	if !s.device.Enabled {
		s.log.Warnf("设备 %s 未启用", s.device.Name)
		return nil
	}

	// 创建可取消的上下文
	ctx, s.cancel = context.WithCancel(ctx)

	s.log.Infof("启动设备服务: %s (%s)", s.device.Name, s.device.IP)

	// 启动Ping检测
	if s.device.Ping.Enabled {
		interval := time.Duration(s.device.Ping.Interval) * time.Second
		s.wg.Add(1)
		go s.pingLoop(ctx, interval)
		s.log.Infof("Ping服务已启动，间隔: %v", interval)
	}

	// 启动Bemfa服务（如果启用）
	if s.device.Bemfa.Enabled {
		s.wg.Add(1)
		go s.bemfaLoop(ctx)
		s.log.Info("Bemfa服务已启动")
	}

	// 启动定时任务
	s.cron.Start()

	return nil
}

// Stop 停止设备服务
func (s *DeviceService) Stop() {
	s.log.Infof("停止设备服务: %s", s.device.Name)

	if s.cancel != nil {
		s.cancel()
	}

	// 停止定时任务
	if s.cron != nil {
		s.cron.Stop()
	}

	// 等待所有goroutine结束
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.log.Infof("设备服务已停止: %s", s.device.Name)
	case <-time.After(5 * time.Second):
		s.log.Warnf("设备服务停止超时: %s", s.device.Name)
	}
}

// pingLoop Ping检测循环
func (s *DeviceService) pingLoop(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 立即执行一次
	s.checkStatus()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkStatus()
		}
	}
}

// checkStatus 检查设备状态
func (s *DeviceService) checkStatus() {
	online, err := core.Ping(s.device.IP, 3*time.Second)
	if err != nil {
		s.log.Debugf("Ping失败: %v", err)
	}

	oldStatus := s.GetStatus()
	var newStatus string
	if online {
		newStatus = "online"
	} else {
		newStatus = "offline"
	}

	s.SetStatus(newStatus)

	// 状态变化时记录日志
	if oldStatus != newStatus && oldStatus != "unknown" {
		statusText := map[string]string{
			"online":  "在线",
			"offline": "离线",
		}
		s.log.Infof("设备状态变化: %s -> %s", statusText[oldStatus], statusText[newStatus])
	}
}

// bemfaLoop Bemfa服务循环
func (s *DeviceService) bemfaLoop(ctx context.Context) {
	defer s.wg.Done()

	// 创建Bemfa客户端
	s.bemfaClient = bemfa.NewClient(s.device, func(cmd string) error {
		switch cmd {
		case "on":
			return s.WakeUp()
		case "off":
			return s.Shutdown()
		}
		return nil
	})

	// 启动Bemfa客户端
	if err := s.bemfaClient.Start(ctx); err != nil {
		s.log.Errorf("启动Bemfa客户端失败: %v", err)
		return
	}

	<-ctx.Done()

	// 停止Bemfa客户端
	if s.bemfaClient != nil {
		s.bemfaClient.Stop()
	}
}

// GetStatus 获取设备状态
func (s *DeviceService) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// SetStatus 设置设备状态
func (s *DeviceService) SetStatus(status string) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()

	// 同步状态到Bemfa
	if s.bemfaClient != nil {
		_ = s.bemfaClient.UpdateStatus(status)
	}
}

// WakeUp 唤醒设备
func (s *DeviceService) WakeUp() error {
	if !s.device.WOL.Enabled {
		return fmt.Errorf("网络唤醒未启用")
	}

	s.log.Infof("发送唤醒指令: %s (MAC: %s)", s.device.Name, s.device.WOL.MAC)

	err := core.WakeOnLAN(
		s.device.WOL.MAC,
		s.device.WOL.Destination,
		s.device.WOL.Port,
		s.device.WOL.Interface,
	)

	if err != nil {
		s.log.Errorf("唤醒失败: %v", err)
		return err
	}

	s.log.Info("唤醒指令已发送")
	return nil
}

// Shutdown 关闭设备
func (s *DeviceService) Shutdown() error {
	if !s.device.Shutdown.Enabled {
		return fmt.Errorf("远程关机未启用")
	}

	s.log.Infof("发送关机指令: %s", s.device.Name)

	result, err := core.Shutdown(
		s.device.IP,
		s.device.Shutdown.Method,
		s.device.Shutdown.Account,
		s.device.Shutdown.Password,
		s.device.Shutdown.Command,
		s.device.Shutdown.Time,
		s.device.Shutdown.Timeout,
	)

	if err != nil {
		s.log.Errorf("关机失败: %v, 输出: %s", err, result)
		return err
	}

	s.log.Infof("关机指令已发送: %s", result)
	return nil
}

// GetDevice 获取设备配置
func (s *DeviceService) GetDevice() *config.Device {
	return s.device
}

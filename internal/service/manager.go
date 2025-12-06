package service

import (
	"context"
	"fmt"
	"powercontrol/internal/config"
	"powercontrol/internal/logger"
	"sync"

	"go.uber.org/zap"
)

// Manager 服务管理器
type Manager struct {
	cfg      *config.Config
	services map[string]*DeviceService
	mu       sync.RWMutex
	log      *zap.SugaredLogger
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewManager 创建服务管理器
func NewManager(cfg *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		cfg:      cfg,
		services: make(map[string]*DeviceService),
		log:      logger.GetLogger("manager"),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// StartAll 启动所有设备服务
func (m *Manager) StartAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Info("启动所有设备服务...")

	for _, device := range m.cfg.Devices {
		if err := m.startDevice(device); err != nil {
			m.log.Errorf("启动设备服务失败 [%s]: %v", device.Name, err)
		}
	}

	m.log.Infof("已启动 %d 个设备服务", len(m.services))
	return nil
}

// StopAll 停止所有设备服务
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.log.Info("停止所有设备服务...")

	// 取消上下文
	if m.cancel != nil {
		m.cancel()
	}

	// 停止所有服务
	for id, svc := range m.services {
		m.log.Infof("停止设备服务: %s", id)
		svc.Stop()
	}

	m.services = make(map[string]*DeviceService)
	m.log.Info("所有设备服务已停止")
}

// StartDevice 启动指定设备服务
func (m *Manager) StartDevice(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 查找设备
	var device *config.Device
	for _, d := range m.cfg.Devices {
		if d.ID == deviceID {
			device = d
			break
		}
	}

	if device == nil {
		return fmt.Errorf("设备不存在: %s", deviceID)
	}

	return m.startDevice(device)
}

// startDevice 启动设备服务（内部方法，需要持有锁）
func (m *Manager) startDevice(device *config.Device) error {
	// 检查是否已存在
	if _, exists := m.services[device.ID]; exists {
		return fmt.Errorf("设备服务已存在: %s", device.ID)
	}

	// 创建并启动服务
	svc := NewDeviceService(device, m.cfg)
	if err := svc.Start(m.ctx); err != nil {
		return err
	}

	m.services[device.ID] = svc
	m.log.Infof("设备服务已启动: %s", device.Name)

	return nil
}

// StopDevice 停止指定设备服务
func (m *Manager) StopDevice(deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	svc, exists := m.services[deviceID]
	if !exists {
		return fmt.Errorf("设备服务不存在: %s", deviceID)
	}

	svc.Stop()
	delete(m.services, deviceID)

	m.log.Infof("设备服务已停止: %s", deviceID)
	return nil
}

// RestartDevice 重启指定设备服务
func (m *Manager) RestartDevice(deviceID string) error {
	m.log.Infof("重启设备服务: %s", deviceID)

	if err := m.StopDevice(deviceID); err != nil {
		return err
	}

	return m.StartDevice(deviceID)
}

// GetService 获取设备服务
func (m *Manager) GetService(deviceID string) (*DeviceService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	svc, exists := m.services[deviceID]
	if !exists {
		return nil, fmt.Errorf("设备服务不存在: %s", deviceID)
	}

	return svc, nil
}

// GetServiceByAlias 通过别名获取设备服务
func (m *Manager) GetServiceByAlias(alias string) (*DeviceService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, svc := range m.services {
		if svc.device.Alias == alias {
			return svc, nil
		}
	}

	return nil, fmt.Errorf("未找到别名为 %s 的设备", alias)
}

// ListServices 列出所有设备服务
func (m *Manager) ListServices() []*DeviceService {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services := make([]*DeviceService, 0, len(m.services))
	for _, svc := range m.services {
		services = append(services, svc)
	}

	return services
}

// IsRunning 检查设备服务是否运行中
func (m *Manager) IsRunning(deviceID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.services[deviceID]
	return exists
}

// AddDevice 添加新设备
func (m *Manager) AddDevice(device *config.Device) error {
	// 添加到配置
	m.cfg.Devices = append(m.cfg.Devices, device)

	// 保存配置
	if err := config.SaveDevice(device); err != nil {
		return fmt.Errorf("保存设备配置失败: %w", err)
	}

	// 启动服务
	return m.StartDevice(device.ID)
}

// RemoveDevice 删除设备
func (m *Manager) RemoveDevice(deviceID string) error {
	// 停止服务
	if err := m.StopDevice(deviceID); err != nil {
		m.log.Warnf("停止设备服务失败: %v", err)
	}

	// 从配置中删除
	for i, d := range m.cfg.Devices {
		if d.ID == deviceID {
			m.cfg.Devices = append(m.cfg.Devices[:i], m.cfg.Devices[i+1:]...)
			break
		}
	}

	// 删除配置文件
	if err := config.DeleteDevice(deviceID); err != nil {
		return fmt.Errorf("删除设备配置失败: %w", err)
	}

	m.log.Infof("设备已删除: %s", deviceID)
	return nil
}

// UpdateConfig 更新配置
func (m *Manager) UpdateConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = cfg
	m.log.Info("配置已更新")
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 全局配置
type Config struct {
	Web     WebConfig     `yaml:"web"`
	Log     LogConfig     `yaml:"log"`
	Message MessageConfig `yaml:"message"`
	Devices []*Device     `yaml:"devices"`
}

// WebConfig Web服务配置
type WebConfig struct {
	Port int    `yaml:"port"`
	Key  string `yaml:"key"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `yaml:"level"`
	KeepDays int    `yaml:"keep_days"`
}

// MessageConfig 消息推送配置
type MessageConfig struct {
	Enabled bool              `yaml:"enabled"`
	Webhook WebhookConfig     `yaml:"webhook"`
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"`
	Method  string `yaml:"method"` // POST, GET
	Headers map[string]string `yaml:"headers"`
}

// Device 设备配置
type Device struct {
	ID       string         `yaml:"id"`
	Name     string         `yaml:"name"`
	Alias    string         `yaml:"alias"`
	IP       string         `yaml:"ip"`
	Enabled  bool           `yaml:"enabled"`
	WOL      WOLConfig      `yaml:"wol"`
	Shutdown ShutdownConfig `yaml:"shutdown"`
	Ping     PingConfig     `yaml:"ping"`
	Bemfa    BemfaConfig    `yaml:"bemfa"`
	Message  DeviceMessage  `yaml:"message"`
}

// WOLConfig 网络唤醒配置
type WOLConfig struct {
	Enabled   bool   `yaml:"enabled"`
	MAC       string `yaml:"mac"`
	Netmask   string `yaml:"netmask"`   // 子网掩码，如 255.255.255.0
	Port      int    `yaml:"port"`
	Interface string `yaml:"interface"`
}

// ShutdownConfig 关机配置
type ShutdownConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Method   string `yaml:"method"` // ssh, smb
	Account  string `yaml:"account"`
	Password string `yaml:"password"`
	Command  string `yaml:"command"`
	Time     int    `yaml:"time"`
	Timeout  int    `yaml:"timeout"`
}

// PingConfig Ping配置
type PingConfig struct {
	Enabled  bool `yaml:"enabled"`
	Interval int  `yaml:"interval"` // 秒
}

// BemfaConfig 巴法云配置
type BemfaConfig struct {
	Enabled bool   `yaml:"enabled"`
	UID     string `yaml:"uid"`
	Topic   string `yaml:"topic"`
}

// DeviceMessage 设备消息配置
type DeviceMessage struct {
	Enabled bool `yaml:"enabled"`
}

var (
	dataDir    = "/app/data"
	defaultDir = "/app/default"
)

// Load 加载配置
func Load() (*Config, error) {
	// 从环境变量读取
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		dataDir = dir
	}

	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 主配置文件路径
	mainFile := filepath.Join(dataDir, "main.yaml")

	// 如果主配置不存在，从默认配置复制
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		if err := copyDefaultConfig(); err != nil {
			return nil, fmt.Errorf("复制默认配置失败: %w", err)
		}
	}

	// 读取主配置
	data, err := os.ReadFile(mainFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量覆盖配置
	if port := os.Getenv("WEB_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Web.Port = p
		}
	}
	if key := os.Getenv("WEB_KEY"); key != "" {
		cfg.Web.Key = key
	}

	// 设置默认值
	if cfg.Web.Port == 0 {
		cfg.Web.Port = 7678
	}
	if cfg.Web.Key == "" {
		cfg.Web.Key = "admin"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "INFO"
	}
	if cfg.Log.KeepDays == 0 {
		cfg.Log.KeepDays = 7
	}

	// 加载设备配置
	if err := loadDevices(cfg); err != nil {
		return nil, fmt.Errorf("加载设备配置失败: %w", err)
	}

	return cfg, nil
}

// Save 保存配置
func Save(cfg *Config) error {
	mainFile := filepath.Join(dataDir, "main.yaml")

	// 保存主配置（不包含设备）
	mainCfg := &Config{
		Web:     cfg.Web,
		Log:     cfg.Log,
		Message: cfg.Message,
	}

	data, err := yaml.Marshal(mainCfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(mainFile, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}

	// 保存设备配置
	for _, device := range cfg.Devices {
		if err := SaveDevice(device); err != nil {
			return err
		}
	}

	return nil
}

// SaveDevice 保存设备配置
func SaveDevice(device *Device) error {
	deviceFile := filepath.Join(dataDir, fmt.Sprintf("device_%s.yaml", device.ID))

	data, err := yaml.Marshal(device)
	if err != nil {
		return fmt.Errorf("序列化设备配置失败: %w", err)
	}

	if err := os.WriteFile(deviceFile, data, 0644); err != nil {
		return fmt.Errorf("保存设备配置失败: %w", err)
	}

	return nil
}

// DeleteDevice 删除设备配置
func DeleteDevice(deviceID string) error {
	deviceFile := filepath.Join(dataDir, fmt.Sprintf("device_%s.yaml", deviceID))
	if err := os.Remove(deviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除设备配置失败: %w", err)
	}
	return nil
}

// loadDevices 加载所有设备配置
func loadDevices(cfg *Config) error {
	files, err := filepath.Glob(filepath.Join(dataDir, "device_*.yaml"))
	if err != nil {
		return err
	}

	cfg.Devices = make([]*Device, 0)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		device := &Device{}
		if err := yaml.Unmarshal(data, device); err != nil {
			continue
		}

		cfg.Devices = append(cfg.Devices, device)
	}

	return nil
}

// copyDefaultConfig 复制默认配置
func copyDefaultConfig() error {
	defaultFile := filepath.Join(defaultDir, "main.yaml")
	mainFile := filepath.Join(dataDir, "main.yaml")

	// 如果默认配置不存在，创建一个基本配置
	if _, err := os.Stat(defaultFile); os.IsNotExist(err) {
		defaultCfg := &Config{
			Web: WebConfig{
				Port: 7678,
				Key:  "admin",
			},
			Log: LogConfig{
				Level:    "INFO",
				KeepDays: 7,
			},
			Message: MessageConfig{
				Enabled: false,
				Webhook: WebhookConfig{
					Enabled: false,
					Method:  "POST",
				},
			},
		}

		data, err := yaml.Marshal(defaultCfg)
		if err != nil {
			return err
		}

		return os.WriteFile(mainFile, data, 0644)
	}

	// 复制默认配置
	data, err := os.ReadFile(defaultFile)
	if err != nil {
		return err
	}

	return os.WriteFile(mainFile, data, 0644)
}

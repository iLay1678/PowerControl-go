package web

import (
	"fmt"
	"net/http"
	"os"
	"powercontrol/internal/config"
	"powercontrol/internal/service"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleWOL 网络唤醒
func (s *Server) handleWOL(c *gin.Context) {
	deviceID := c.Param("device")
	clientIP := c.ClientIP()

	// 获取设备服务
	svc, err := s.getServiceByIDOrAlias(deviceID)
	if err != nil {
		s.log.Warnf("IP=%s 访问WOL API失败: %v", clientIP, err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	device := svc.GetDevice()
	s.log.Debugf("IP=%s 访问WOL API: %s", clientIP, device.Name)

	// 执行唤醒
	err = svc.WakeUp()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"device_name": device.Name,
			"device_ip":   device.IP,
			"method":      "wol",
			"message":     "error",
			"message_cn":  "发送唤醒指令失败",
			"result":      err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_name": device.Name,
		"device_ip":   device.IP,
		"method":      "wol",
		"message":     "success",
		"message_cn":  "已发送唤醒指令",
		"result":      "done",
	})
}

// handleShutdown 远程关机
func (s *Server) handleShutdown(c *gin.Context) {
	deviceID := c.Param("device")
	clientIP := c.ClientIP()

	svc, err := s.getServiceByIDOrAlias(deviceID)
	if err != nil {
		s.log.Warnf("IP=%s 访问关机API失败: %v", clientIP, err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	device := svc.GetDevice()
	s.log.Debugf("IP=%s 访问关机API: %s", clientIP, device.Name)

	err = svc.Shutdown()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"device_name": device.Name,
			"device_ip":   device.IP,
			"method":      "shutdown",
			"message":     "error",
			"message_cn":  "发送关机指令失败",
			"result":      err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_name": device.Name,
		"device_ip":   device.IP,
		"method":      "shutdown",
		"message":     "success",
		"message_cn":  "已发送关机指令",
		"result":      "succeeded",
	})
}

// handlePing Ping检测
func (s *Server) handlePing(c *gin.Context) {
	deviceID := c.Param("device")
	clientIP := c.ClientIP()

	svc, err := s.getServiceByIDOrAlias(deviceID)
	if err != nil {
		s.log.Warnf("IP=%s 访问Ping API失败: %v", clientIP, err)
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	device := svc.GetDevice()
	s.log.Infof("IP=%s 访问Ping API: 设备=%s, IP=%s", clientIP, device.Name, device.IP)

	status := svc.GetStatus()

	statusMap := map[string]string{
		"online":  "在线",
		"offline": "离线",
		"unknown": "未知",
	}

	s.log.Infof("Ping检测结果: 设备=%s, 状态=%s", device.Name, statusMap[status])

	c.JSON(http.StatusOK, gin.H{
		"device_name":      device.Name,
		"device_ip":        device.IP,
		"method":           "ping",
		"device_status":    status,
		"device_status_cn": statusMap[status],
		"ping_result":      status,
	})
}

// handleGetAllDevices 获取所有设备信息
func (s *Server) handleGetAllDevices(c *gin.Context) {
	services := s.svcMgr.ListServices()

	result := make(map[string]interface{})
	result["main"] = gin.H{
		"log_level":       s.cfg.Log.Level,
		"log_days":        s.cfg.Log.KeepDays,
		"message_enabled": s.cfg.Message.Enabled,
	}

	for _, svc := range services {
		device := svc.GetDevice()
		result[device.ID] = gin.H{
			"enabled": device.Enabled,
			"name":    device.Name,
			"alias":   device.Alias,
			"ip":      device.IP,
			"running": s.svcMgr.IsRunning(device.ID),
			"status":  []string{svc.GetStatus()},
		}
	}

	c.JSON(http.StatusOK, result)
}

// handleGetDevice 获取设备详情
func (s *Server) handleGetDevice(c *gin.Context) {
	deviceID := c.Param("id")

	svc, err := s.svcMgr.GetService(deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "设备不存在",
		})
		return
	}

	device := svc.GetDevice()
	c.JSON(http.StatusOK, gin.H{
		"device_id":      device.ID,
		"device_yaml":    device,
		"device_status":  []string{svc.GetStatus()},
		"device_running": s.svcMgr.IsRunning(device.ID),
	})
}

// handleNewDevice 创建新设备
func (s *Server) handleNewDevice(c *gin.Context) {
	// 接收请求数据
	var device config.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":    false,
			"device_id": "",
			"message":   "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 生成设备ID
	device.ID = "device_" + uuid.New().String()[:8]

	// 添加设备
	if err := s.svcMgr.AddDevice(&device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result":    false,
			"device_id": "",
			"message":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result":    true,
		"device_id": device.ID,
	})
}

// handleUpdateDevice 更新设备配置
func (s *Server) handleUpdateDevice(c *gin.Context) {
	deviceID := c.Param("id")

	var device config.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": "无效的请求数据",
		})
		return
	}

	// 确保设备ID一致
	device.ID = deviceID

	// 保存配置
	if err := config.SaveDevice(&device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}

	// 重新加载配置
	cfg, err := config.Load()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result":  false,
			"message": "重新加载配置失败: " + err.Error(),
		})
		return
	}
	s.cfg = cfg

	// 更新 Manager 的配置引用
	s.svcMgr.UpdateConfig(cfg)

	// 重启设备服务以应用新配置
	if s.svcMgr.IsRunning(deviceID) {
		if err := s.svcMgr.RestartDevice(deviceID); err != nil {
			s.log.Warnf("重启设备服务失败: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"result":    true,
	})
}

// handleDeleteDevice 删除设备
func (s *Server) handleDeleteDevice(c *gin.Context) {
	deviceID := c.Param("id")

	if err := s.svcMgr.RemoveDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"result":    true,
	})
}

// handleStartService 启动设备服务
func (s *Server) handleStartService(c *gin.Context) {
	deviceID := c.Param("id")

	if err := s.svcMgr.StartDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"device_id": deviceID,
			"result":    false,
			"message":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"result":    true,
	})
}

// handleStopService 停止设备服务
func (s *Server) handleStopService(c *gin.Context) {
	deviceID := c.Param("id")

	if err := s.svcMgr.StopDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"device_id": deviceID,
			"result":    false,
			"message":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"result":    true,
	})
}

// handleRestartService 重启设备服务
func (s *Server) handleRestartService(c *gin.Context) {
	deviceID := c.Param("id")

	if err := s.svcMgr.RestartDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"device_id": deviceID,
			"result":    false,
			"message":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"result":    true,
	})
}

// handleWOLAll 全部唤醒
func (s *Server) handleWOLAll(c *gin.Context) {
	services := s.svcMgr.ListServices()
	results := make(map[string]interface{})

	for _, svc := range services {
		device := svc.GetDevice()
		err := svc.WakeUp()

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "已发送唤醒指令"
		} else if err.Error() == "网络唤醒未启用" {
			message = "skip"
			messageCN = "网络唤醒未启用，跳过"
		} else {
			message = "error"
			messageCN = "发送唤醒指令失败"
		}

		results[device.ID] = gin.H{
			"name":       device.Name,
			"device_ip":  device.IP,
			"method":     "wol",
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleShutdownAll 全部关机
func (s *Server) handleShutdownAll(c *gin.Context) {
	services := s.svcMgr.ListServices()
	results := make(map[string]interface{})

	for _, svc := range services {
		device := svc.GetDevice()
		err := svc.Shutdown()

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "已发送关机指令"
		} else if err.Error() == "远程关机未启用" {
			message = "skip"
			messageCN = "远程关机未启用，跳过"
		} else {
			message = "error"
			messageCN = "发送关机指令失败"
		}

		results[device.ID] = gin.H{
			"name":       device.Name,
			"device_ip":  device.IP,
			"method":     "shutdown",
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleBatchWOL 批量唤醒
func (s *Server) handleBatchWOL(c *gin.Context) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	results := make(map[string]interface{})
	for _, deviceID := range req.DeviceIDs {
		svc, err := s.svcMgr.GetService(deviceID)
		if err != nil {
			continue
		}

		device := svc.GetDevice()
		err = svc.WakeUp()

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "已发送唤醒指令"
		} else {
			message = "error"
			messageCN = "发送唤醒指令失败"
		}

		results[deviceID] = gin.H{
			"name":       device.Name,
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleBatchShutdown 批量关机
func (s *Server) handleBatchShutdown(c *gin.Context) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	results := make(map[string]interface{})
	for _, deviceID := range req.DeviceIDs {
		svc, err := s.svcMgr.GetService(deviceID)
		if err != nil {
			continue
		}

		device := svc.GetDevice()
		err = svc.Shutdown()

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "已发送关机指令"
		} else {
			message = "error"
			messageCN = "发送关机指令失败"
		}

		results[deviceID] = gin.H{
			"name":       device.Name,
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleBatchStart 批量启动服务
func (s *Server) handleBatchStart(c *gin.Context) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	results := make(map[string]interface{})
	for _, deviceID := range req.DeviceIDs {
		err := s.svcMgr.StartDevice(deviceID)

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "服务已启动"
		} else {
			message = "error"
			messageCN = "服务启动失败"
		}

		results[deviceID] = gin.H{
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleBatchStop 批量停止服务
func (s *Server) handleBatchStop(c *gin.Context) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	results := make(map[string]interface{})
	for _, deviceID := range req.DeviceIDs {
		err := s.svcMgr.StopDevice(deviceID)

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "服务已停止"
		} else {
			message = "error"
			messageCN = "服务停止失败"
		}

		results[deviceID] = gin.H{
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleBatchRestart 批量重启服务
func (s *Server) handleBatchRestart(c *gin.Context) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	results := make(map[string]interface{})
	for _, deviceID := range req.DeviceIDs {
		err := s.svcMgr.RestartDevice(deviceID)

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "服务已重启"
		} else {
			message = "error"
			messageCN = "服务重启失败"
		}

		results[deviceID] = gin.H{
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleBatchDelete 批量删除设备
func (s *Server) handleBatchDelete(c *gin.Context) {
	var req struct {
		DeviceIDs []string `json:"device_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	results := make(map[string]interface{})
	for _, deviceID := range req.DeviceIDs {
		err := s.svcMgr.RemoveDevice(deviceID)

		var message, messageCN string
		if err == nil {
			message = "success"
			messageCN = "设备已删除"
		} else {
			message = "error"
			messageCN = "设备删除失败"
		}

		results[deviceID] = gin.H{
			"message":    message,
			"message_cn": messageCN,
		}
	}

	c.JSON(http.StatusOK, results)
}

// handleGetConfig 获取配置
func (s *Server) handleGetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, s.cfg)
}

// handleUpdateConfig 更新配置
func (s *Server) handleUpdateConfig(c *gin.Context) {
	var cfg config.Config
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求数据",
		})
		return
	}

	if err := config.Save(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置已保存",
	})
}

// handleGetLogs 获取日志列表
func (s *Server) handleGetLogs(c *gin.Context) {
	// 简化实现，仅返回日志文件列表
	c.JSON(http.StatusOK, gin.H{
		"message": "日志功能待实现",
	})
}

// handleGetMainBasic 获取主程序基本信息
func (s *Server) handleGetMainBasic(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": "4.0.0",
		"run_time": fmt.Sprintf("运行中: %s", time.Now().Format("2006-01-02 15:04:05")),
	})
}

// handleRestart 重启程序
func (s *Server) handleRestart(c *gin.Context) {
	s.log.Warn("收到重启请求，程序即将退出...")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "程序正在重启...",
	})

	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()
}

// getServiceByIDOrAlias 通过ID或别名获取服务
func (s *Server) getServiceByIDOrAlias(idOrAlias string) (*service.DeviceService, error) {
	// 先尝试作为ID查找
	svc, err := s.svcMgr.GetService(idOrAlias)
	if err == nil {
		return svc, nil
	}

	// 尝试作为别名查找（空格转短横线）
	alias := strings.ReplaceAll(idOrAlias, " ", "-")
	svc, err = s.svcMgr.GetServiceByAlias(alias)
	if err == nil {
		return svc, nil
	}

	return nil, fmt.Errorf("设备不存在: %s", idOrAlias)
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"powercontrol/internal/config"
	"powercontrol/internal/logger"
	"powercontrol/internal/service"
	"powercontrol/internal/web"
	"syscall"
	"time"
)

const Version = "4.0.0"

func main() {
	// 打印启动信息
	fmt.Printf("PowerControl v%s 正在启动...\n", Version)

	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志系统
	if err := logger.Init(cfg.Log.Level, cfg.Log.KeepDays); err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	log := logger.GetLogger("main")
	log.Info("PowerControl 初始化完成")
	log.Infof("版本: %s", Version)
	log.Infof("Web端口: %d", cfg.Web.Port)
	log.Infof("日志级别: %s", cfg.Log.Level)

	// 创建服务管理器
	svcMgr := service.NewManager(cfg)

	// 启动所有设备服务
	if err := svcMgr.StartAll(); err != nil {
		log.Errorf("启动设备服务失败: %v", err)
	}

	// 启动Web服务
	webServer := web.NewServer(cfg, svcMgr)
	go func() {
		if err := webServer.Start(); err != nil {
			log.Errorf("Web服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	log.Infof("PowerControl 已启动，访问地址: http://0.0.0.0:%d", cfg.Web.Port)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭 PowerControl...")

	// 优雅停止Web服务
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := webServer.Shutdown(ctx); err != nil {
		log.Errorf("Web服务关闭失败: %v", err)
	}

	// 停止所有设备服务
	svcMgr.StopAll()

	log.Info("PowerControl 已停止")
}

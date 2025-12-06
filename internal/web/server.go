package web

import (
	"context"
	"fmt"
	"net/http"
	"powercontrol/internal/config"
	"powercontrol/internal/logger"
	"powercontrol/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server Web服务器
type Server struct {
	cfg    *config.Config
	svcMgr *service.Manager
	engine *gin.Engine
	server *http.Server
	log    *zap.SugaredLogger
}

// NewServer 创建Web服务器
func NewServer(cfg *config.Config, svcMgr *service.Manager) *Server {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		cfg:    cfg,
		svcMgr: svcMgr,
		engine: gin.New(),
		log:    logger.GetLogger("web"),
	}

	// 配置中间件
	s.engine.Use(gin.Recovery())
	s.engine.Use(s.loggerMiddleware())
	s.engine.Use(s.corsMiddleware())

	// 配置路由
	s.setupRoutes()

	return s
}

// Start 启动Web服务
func (s *Server) Start() error {
	addr := fmt.Sprintf("0.0.0.0:%d", s.cfg.Web.Port)

	s.server = &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	s.log.Infof("Web服务启动: %s", addr)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("Web服务启动失败: %w", err)
	}

	return nil
}

// Shutdown 停止Web服务
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	s.log.Info("正在关闭Web服务...")
	return s.server.Shutdown(ctx)
}

// setupRoutes 配置路由
func (s *Server) setupRoutes() {
	// 静态文件
	s.engine.StaticFile("/", "./web/index.html")
	s.engine.Static("/static", "./web/static")

	// API路由组（需要KEY鉴权）
	api := s.engine.Group("/", s.authMiddleware())
	{
		// 设备操作
		api.GET("/wol/:device", s.handleWOL)
		api.GET("/shutdown/:device", s.handleShutdown)
		api.GET("/ping/:device", s.handlePing)

		// 设备管理
		api.GET("/device/get/all-brief", s.handleGetAllDevices)
		api.GET("/device/get/:id", s.handleGetDevice)
		api.POST("/device/new", s.handleNewDevice)
		api.POST("/device/update/:id", s.handleUpdateDevice)
		api.DELETE("/device/delete/:id", s.handleDeleteDevice)

		// 设备服务控制
		api.POST("/device/start/:id", s.handleStartService)
		api.POST("/device/stop/:id", s.handleStopService)
		api.POST("/device/restart/:id", s.handleRestartService)

		// 批量操作
		api.POST("/device/batch/wol", s.handleBatchWOL)
		api.POST("/device/batch/shutdown", s.handleBatchShutdown)
		api.POST("/device/batch/start", s.handleBatchStart)
		api.POST("/device/batch/stop", s.handleBatchStop)
		api.POST("/device/batch/restart", s.handleBatchRestart)
		api.POST("/device/batch/delete", s.handleBatchDelete)

		// 全部操作
		api.POST("/device/wol/all", s.handleWOLAll)
		api.POST("/device/shutdown/all", s.handleShutdownAll)

		// 配置管理
		api.GET("/config", s.handleGetConfig)
		api.POST("/config", s.handleUpdateConfig)

		// 日志
		api.GET("/logs", s.handleGetLogs)

		// 系统信息
		api.GET("/device/get/main-basic", s.handleGetMainBasic)

		// 重启程序
		api.POST("/restart", s.handleRestart)
	}
}

// loggerMiddleware 日志中间件
func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if query != "" {
			path = path + "?" + query
		}

		s.log.Debugf("[%s] %s %s %d %v", clientIP, method, path, statusCode, duration)
	}
}

// corsMiddleware CORS中间件
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// authMiddleware 鉴权中间件
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Query("key")
		if key != s.cfg.Web.Key {
			clientIP := c.ClientIP()
			s.log.Warnf("鉴权失败: IP=%s, KEY=%s", clientIP, key)
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "KEY不正确",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// handleIndex 首页
func (s *Server) handleIndex(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "PowerControl",
		"version": "4.0.0",
		"message": "服务运行中",
	})
}

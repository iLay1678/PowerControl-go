package bemfa

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"powercontrol/internal/config"
	"powercontrol/internal/logger"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	BemfaServer = "bemfa.com:8344"
	HeartbeatInterval = 60 * time.Second
	ReconnectDelay = 60 * time.Second
	MaxReconnectPerDay = 5
)

// Client Bemfa客户端
type Client struct {
	device    *config.Device
	log       *zap.SugaredLogger
	conn      net.Conn
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	reconnectCount int
	onCommand func(cmd string) error
}

// NewClient 创建Bemfa客户端
func NewClient(device *config.Device, onCommand func(cmd string) error) *Client {
	return &Client{
		device:    device,
		log:       logger.GetLogger(fmt.Sprintf("bemfa_%s", device.Name)),
		onCommand: onCommand,
	}
}

// Start 启动Bemfa客户端
func (c *Client) Start(ctx context.Context) error {
	if !c.device.Bemfa.Enabled {
		return nil
	}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.log.Info("启动Bemfa服务")

	// 启动连接循环
	c.wg.Add(1)
	go c.connectionLoop()

	// 启动每日重置定时器
	c.wg.Add(1)
	go c.dailyReset()

	return nil
}

// Stop 停止Bemfa客户端
func (c *Client) Stop() {
	c.log.Info("停止Bemfa服务")

	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		c.conn.Close()
	}

	c.wg.Wait()
}

// connectionLoop 连接循环
func (c *Client) connectionLoop() {
	defer c.wg.Done()

	retry := false

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// 重连延迟
		if retry {
			c.reconnectCount++
			c.log.Warnf("正在重新连接Bemfa (今日第%d次)", c.reconnectCount)

			delay := time.Second * 5
			if c.reconnectCount > MaxReconnectPerDay {
				delay = ReconnectDelay
				if c.reconnectCount == MaxReconnectPerDay+1 {
					c.log.Warn("今日重连已达5次，后续重连将有60秒等待时间")
				}
			}

			select {
			case <-c.ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		// 连接服务器
		if err := c.connect(); err != nil {
			c.log.Errorf("连接Bemfa失败: %v", err)
			retry = true
			continue
		}

		// 订阅主题
		if err := c.subscribe(); err != nil {
			c.log.Errorf("订阅Bemfa主题失败: %v", err)
			retry = true
			continue
		}

		c.log.Info("Bemfa订阅成功")

		// 启动心跳
		heartbeatCtx, heartbeatCancel := context.WithCancel(c.ctx)
		go c.heartbeatLoop(heartbeatCtx)

		// 启动状态更新
		statusCtx, statusCancel := context.WithCancel(c.ctx)
		go c.statusUpdateLoop(statusCtx)

		// 处理消息
		c.handleMessages()

		// 清理
		heartbeatCancel()
		statusCancel()

		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}

		retry = true
	}
}

// connect 连接到Bemfa服务器
func (c *Client) connect() error {
	conn, err := net.DialTimeout("tcp", BemfaServer, 10*time.Second)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// subscribe 订阅主题
func (c *Client) subscribe() error {
	msg := fmt.Sprintf("cmd=1&uid=%s&topic=%s\r\n",
		c.device.Bemfa.UID,
		c.device.Bemfa.Topic)

	_, err := c.conn.Write([]byte(msg))
	return err
}

// heartbeatLoop 心跳循环
func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				c.log.Errorf("发送心跳失败: %v", err)
			} else {
				c.log.Debug("Bemfa心跳已发送")
			}
		}
	}
}

// sendHeartbeat 发送心跳
func (c *Client) sendHeartbeat() error {
	if c.conn == nil {
		return fmt.Errorf("连接未建立")
	}

	_, err := c.conn.Write([]byte("ping\r\n"))
	return err
}

// statusUpdateLoop 状态更新循环
func (c *Client) statusUpdateLoop(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// 延迟5秒后开始第一次更新
	time.Sleep(5 * time.Second)
	c.updateStatus("unknown")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 状态由外部设置，这里只负责同步
		}
	}
}

// UpdateStatus 更新设备状态到Bemfa
func (c *Client) UpdateStatus(status string) error {
	if c.conn == nil || !c.device.Bemfa.Enabled {
		return nil
	}

	return c.updateStatus(status)
}

// updateStatus 更新状态到Bemfa
func (c *Client) updateStatus(status string) error {
	// 将状态映射为on/off
	var state string
	switch status {
	case "online":
		state = "on"
	case "offline":
		state = "off"
	default:
		return nil
	}

	msg := fmt.Sprintf("cmd=2&uid=%s&topic=%s/up&msg=%s\r\n",
		c.device.Bemfa.UID,
		c.device.Bemfa.Topic,
		state)

	if c.conn == nil {
		return fmt.Errorf("连接未建立")
	}

	_, err := c.conn.Write([]byte(msg))
	if err == nil {
		c.log.Debugf("Bemfa云端设备状态已更新: %s", status)
	}
	return err
}

// handleMessages 处理接收到的消息
func (c *Client) handleMessages() {
	reader := bufio.NewReader(c.conn)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// 设置读取超时
		c.conn.SetReadDeadline(time.Now().Add(2 * HeartbeatInterval))

		line, err := reader.ReadString('\n')
		if err != nil {
			c.log.Errorf("接收Bemfa消息失败: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		c.log.Debugf("收到Bemfa消息: %s", line)

		// 解析消息
		if strings.Contains(line, fmt.Sprintf("uid=%s", c.device.Bemfa.UID)) {
			if strings.Contains(line, "msg=on") {
				c.log.Info("收到Bemfa开机指令")
				if c.onCommand != nil {
					if err := c.onCommand("on"); err != nil {
						c.log.Errorf("执行开机指令失败: %v", err)
					}
				}
			} else if strings.Contains(line, "msg=off") {
				c.log.Info("收到Bemfa关机指令")
				if c.onCommand != nil {
					if err := c.onCommand("off"); err != nil {
						c.log.Errorf("执行关机指令失败: %v", err)
					}
				}
			}
		}
	}
}

// dailyReset 每日重置重连计数
func (c *Client) dailyReset() {
	defer c.wg.Done()

	for {
		// 计算到下一个0点的时间
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		duration := next.Sub(now)

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(duration):
			c.reconnectCount = 0
			c.log.Info("重连计数已重置")
		}
	}
}

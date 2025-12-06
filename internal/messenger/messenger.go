package messenger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"powercontrol/internal/config"
	"time"
)

// Messenger 消息推送器
type Messenger struct {
	cfg    *config.MessageConfig
	client *http.Client
}

// New 创建消息推送器
func New(cfg *config.MessageConfig) *Messenger {
	return &Messenger{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send 发送消息
func (m *Messenger) Send(title, content string) error {
	if !m.cfg.Enabled {
		return nil
	}

	if m.cfg.Webhook.Enabled {
		return m.sendWebhook(title, content)
	}

	return nil
}

// sendWebhook 发送Webhook消息
func (m *Messenger) sendWebhook(title, content string) error {
	if m.cfg.Webhook.URL == "" {
		return fmt.Errorf("Webhook URL未配置")
	}

	// 构造消息体
	payload := map[string]interface{}{
		"title":   title,
		"content": content,
		"time":    time.Now().Format("2006-01-02 15:04:05"),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 创建请求
	method := m.cfg.Webhook.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, m.cfg.Webhook.URL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	for key, value := range m.cfg.Webhook.Headers {
		req.Header.Set(key, value)
	}

	// 发送请求
	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Webhook返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendAsync 异步发送消息（不等待结果）
func (m *Messenger) SendAsync(title, content string) {
	go func() {
		if err := m.Send(title, content); err != nil {
			// 记录错误但不阻塞
			fmt.Printf("发送消息失败: %v\n", err)
		}
	}()
}

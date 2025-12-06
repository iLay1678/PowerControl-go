package core

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Shutdown 远程关机
func Shutdown(ip, method, account, password, command string, delay, timeout int) (string, error) {
	switch method {
	case "ssh":
		return shutdownViaSSH(ip, account, password, command, delay, timeout)
	case "smb":
		return shutdownViaSMB(ip, account, password, delay, timeout)
	case "custom":
		return executeCustomCommand(command)
	default:
		return "", fmt.Errorf("不支持的关机方法: %s", method)
	}
}

// shutdownViaSSH 通过SSH关机
func shutdownViaSSH(ip, account, password, command string, delay, timeout int) (string, error) {
	if command == "" {
		command = fmt.Sprintf("sleep %d && sudo shutdown -h now", delay)
	}

	// 使用sshpass执行SSH命令
	cmd := exec.Command("sshpass", "-p", password, "ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", fmt.Sprintf("ConnectTimeout=%d", timeout),
		fmt.Sprintf("%s@%s", account, ip),
		command,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("SSH关机失败: %w", err)
	}

	return "shutdown command succeeded via SSH", nil
}

// shutdownViaSMB 通过SMB/RPC关机 (Windows)
func shutdownViaSMB(ip, account, password string, delay, timeout int) (string, error) {
	// 使用net rpc命令（Linux）或Windows RPC
	domain := ""
	if strings.Contains(account, "\\") {
		parts := strings.SplitN(account, "\\", 2)
		domain = parts[0]
		account = parts[1]
	}

	args := []string{
		"rpc",
		"shutdown",
		"-I", ip,
		"-U", fmt.Sprintf("%s%%%s", account, password),
		"-t", fmt.Sprintf("%d", delay),
		"-f",
	}

	if domain != "" {
		args = append(args, "-W", domain)
	}

	cmd := exec.Command("net", args...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("SMB_TIMEOUT=%d", timeout))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("SMB关机失败: %w", err)
	}

	if strings.Contains(string(output), "succeeded") || strings.Contains(string(output), "成功") {
		return "shutdown command succeeded via SMB", nil
	}

	return string(output), nil
}

// executeCustomCommand 执行自定义命令
func executeCustomCommand(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("自定义命令不能为空")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return string(output), fmt.Errorf("执行自定义命令失败: %w", err)
	}

	return string(output), nil
}

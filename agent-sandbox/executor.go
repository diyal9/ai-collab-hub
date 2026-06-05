package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/creack/pty"
)

// Executor 接口
type Executor interface {
	Name() string
	Execute(taskID string, payload TaskPayload, client *SandboxClient) error
}

// ShellExecutor 基础 Shell 执行器
type ShellExecutor struct{}

func (e *ShellExecutor) Name() string { return "shell" }

func (e *ShellExecutor) Execute(taskID string, payload TaskPayload, client *SandboxClient) error {
	cmd := exec.Command("bash", "-c", payload.Command)
	cmd.Dir = payload.WorkDir
	return runPTY(taskID, cmd, client)
}

// CursorExecutor Cursor Headless 执行器
type CursorExecutor struct{}

func (e *CursorExecutor) Name() string { return "cursor" }

func (e *CursorExecutor) Execute(taskID string, payload TaskPayload, client *SandboxClient) error {
	// 使用 cursor agent 命令 (根据官方 Headless CLI 文档)
	// 如果 Cursor 路径不在 PATH 中，可能需要指定绝对路径
	cmd := exec.Command("cursor", "agent",
		"--project", payload.WorkDir,
		"--run", payload.Command,
		"--yes", // 自动确认
	)
	cmd.Dir = payload.WorkDir
	return runPTY(taskID, cmd, client)
}

// AiderExecutor Aider CLI 执行器
type AiderExecutor struct{}

func (e *AiderExecutor) Name() string { return "aider" }

func (e *AiderExecutor) Execute(taskID string, payload TaskPayload, client *SandboxClient) error {
	cmd := exec.Command("aider",
		"--yes",
		"--message", payload.Command,
		"--read", // 只读模式，避免意外修改，可配置
	)
	cmd.Dir = payload.WorkDir
	return runPTY(taskID, cmd, client)
}

// runPTY 通用 PTY 执行逻辑
func runPTY(taskID string, cmd *exec.Cmd, client *SandboxClient) error {
	log.Printf("[Executor] Running command for task %s: %s", taskID, cmd.String())

	// 上报准备中
	client.SendStatus(taskID, "preparing")

	// 启动 PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return client.SendError(taskID, fmt.Sprintf("PTY start failed: %v", err))
	}
	defer ptmx.Close()

	// 上报运行中
	client.SendStatus(taskID, "running")

	// 流式读取日志
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			line := string(buf[:n])
			// 简单清理控制字符
			cleanLine := strings.Map(func(r rune) rune {
				if r < 32 && r != '\n' && r != '\r' && r != '\t' {
					return ' '
				}
				return r
			}, line)
			client.SendLog(taskID, cleanLine)
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[Executor] Read error for task %s: %v", taskID, err)
			}
			break
		}
	}

	// 等待进程结束
	state, err := cmd.Process.Wait()
	if err != nil {
		return client.SendError(taskID, fmt.Sprintf("Process wait failed: %v", err))
	}
	if state.Success() {
		return client.SendStatus(taskID, "completed")
	}
	return client.SendError(taskID, fmt.Sprintf("Process exited with code %d", state.ExitCode()))
}

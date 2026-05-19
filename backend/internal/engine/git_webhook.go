package engine

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/model"
)

// ─── Git Webhook 处理器 ───

// GitRepoMapping: 仓库到 Flow 的映射
type GitRepoMapping struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RepoURL   string    `json:"repo_url" gorm:"uniqueIndex"`
	RepoName  string    `json:"repo_name"`
	FlowID    uint      `json:"flow_id"`
	Branch    string    `json:"branch"`         // 触发分支，* 表示所有
	NodeMatch string    `json:"node_match"`     // 节点匹配模式: branch, prefix, tag
	CreatedAt time.Time `json:"created_at"`
}

// GitCommit: 提交记录
type GitCommit struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	RepoURL   string    `json:"repo_url"`
	Branch    string    `json:"branch"`
	CommitSHA string    `json:"commit_sha"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	TaskID    string    `json:"task_id"`         // 关联的 Flow 任务 ID
	Status    string    `json:"status"`          // processed, ignored, failed
	CreatedAt time.Time `json:"created_at"`
}

func AutoMigrateGitWebhook() {
	db.DB.AutoMigrate(&GitRepoMapping{}, &GitCommit{})
}

// HandleGitPush 处理 Git Push 事件
func HandleGitPush(payload map[string]interface{}, eventType string) {
	var repoURL, branch, commitSHA, author, message string

	if eventType == "push" || eventType == "" {
		// GitHub format
		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			if url, ok := repo["html_url"].(string); ok {
				repoURL = url
			}
			if name, ok := repo["full_name"].(string); ok {
				repoURL = name
			}
		}
		ref, _ := payload["ref"].(string)
		branch = strings.TrimPrefix(ref, "refs/heads/")
		
		if head, ok := payload["after"].(string); ok {
			commitSHA = head
		}
		if pusher, ok := payload["pusher"].(map[string]interface{}); ok {
			if name, ok := pusher["name"].(string); ok {
				author = name
			}
		}
		commits, _ := payload["commits"].([]interface{})
		if len(commits) > 0 {
			if c, ok := commits[len(commits)-1].(map[string]interface{}); ok {
				if msg, ok := c["message"].(string); ok {
					message = msg
				}
			}
		}
	} else if payload["object_kind"] == "push" {
		// GitLab format
		if p, ok := payload["project"].(map[string]interface{}); ok {
			if url, ok := p["web_url"].(string); ok {
				repoURL = url
			}
			if name, ok := p["path_with_namespace"].(string); ok {
				repoURL = name
			}
		}
		ref, _ := payload["ref"].(string)
		branch = strings.TrimPrefix(ref, "refs/heads/")
		commitSHA, _ = payload["after"].(string)
		if user, ok := payload["user_name"].(string); ok {
			author = user
		}
		if commits, ok := payload["commits"].([]interface{}); ok && len(commits) > 0 {
			if c, ok := commits[len(commits)-1].(map[string]interface{}); ok {
				if msg, ok := c["message"].(string); ok {
					message = msg
				}
			}
		}
	}

	if repoURL == "" || branch == "" {
		log.Printf("[Webhook] Invalid push payload")
		return
	}

	log.Printf("[Webhook] Push to %s (%s) by %s: %s", repoURL, branch, author, message)

	// 查找匹配的 Repo Mapping
	var mappings []GitRepoMapping
	db.DB.Where("repo_url = ? OR repo_url = ?", repoURL, extractRepoKey(repoURL)).Find(&mappings)

	// 也尝试模糊匹配（部分 URL）
	if len(mappings) == 0 {
		key := extractRepoKey(repoURL)
		db.DB.Where("repo_name LIKE ?", "%"+key+"%").Find(&mappings)
	}

	// 记录提交
	commit := GitCommit{
		RepoURL:   repoURL,
		Branch:    branch,
		CommitSHA: commitSHA,
		Author:    author,
		Message:   message,
		Status:    "processed",
	}
	db.DB.Create(&commit)

	// 处理匹配的 Flow
	for _, mapping := range mappings {
		// 检查分支匹配
		if mapping.Branch != "" && mapping.Branch != "*" && mapping.Branch != branch {
			continue
		}

		log.Printf("[Webhook] Matched flow %d for %s branch %s", mapping.FlowID, repoURL, branch)

		// 根据 node_match 策略处理
		switch mapping.NodeMatch {
		case "branch":
			// 分支名匹配节点名
			matchNodeByBranch(mapping.FlowID, branch, commit)
		case "prefix":
			// 提交消息前缀匹配节点
			matchNodeByPrefix(mapping.FlowID, message, commit)
		case "tag":
			// 标签匹配
			matchNodeByTag(mapping.FlowID, message, branch, commit)
		default:
			// 默认：尝试完成所有运行中的 agent 节点
			completeRunningNodes(mapping.FlowID, commit)
		}
	}
}

// HandlePullRequest 处理 PR/MR 事件
func HandlePullRequest(payload map[string]interface{}, action string, eventType string) {
	var repoURL, branch, title, author string
	var number int

	if eventType == "pull_request" {
		// GitHub
		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			if name, ok := repo["full_name"].(string); ok {
				repoURL = name
			}
		}
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			if head, ok := pr["head"].(map[string]interface{}); ok {
				if ref, ok := head["ref"].(string); ok {
					branch = ref
				}
			}
			if t, ok := pr["title"].(string); ok {
				title = t
			}
			if n, ok := pr["number"].(float64); ok {
				number = int(n)
			}
		}
		if user, ok := payload["sender"].(map[string]interface{}); ok {
			if login, ok := user["login"].(string); ok {
				author = login
			}
		}
	} else {
		// GitLab
		if obj, ok := payload["object_attributes"].(map[string]interface{}); ok {
			if t, ok := obj["title"].(string); ok {
				title = t
			}
			if s, ok := obj["source_branch"].(string); ok {
				branch = s
			}
			if n, ok := obj["iid"].(float64); ok {
				number = int(n)
			}
		}
		if p, ok := payload["project"].(map[string]interface{}); ok {
			if name, ok := p["path_with_namespace"].(string); ok {
				repoURL = name
			}
		}
		if user, ok := payload["user"].(map[string]interface{}); ok {
			if name, ok := user["name"].(string); ok {
				author = name
			}
		}
	}

	log.Printf("[Webhook] PR/MR #%d on %s: %s (action: %s)", number, repoURL, title, action)

	// PR 合并时自动完成关联任务
	if action == "closed" || action == "merge" || action == "accepted" {
		var mappings []GitRepoMapping
		db.DB.Where("repo_url LIKE ?", "%"+extractRepoKey(repoURL)+"%").Find(&mappings)
		for _, m := range mappings {
			completeRunningNodes(m.FlowID, GitCommit{
				RepoURL: repoURL,
				Branch:  branch,
				Message: fmt.Sprintf("PR #%d merged: %s", number, title),
				Author:  author,
			})
		}
	}
}

// matchNodeByBranch 根据分支名匹配节点
func matchNodeByBranch(flowID uint, branch string, commit GitCommit) {
	var nodes []model.AgentNode
	db.DB.Where("flow_id = ?", flowID).Find(&nodes)

	for _, node := range nodes {
		// 检查节点名是否包含分支名
		if strings.Contains(strings.ToLower(node.Name), strings.ToLower(branch)) ||
			strings.Contains(strings.ToLower(branch), strings.ToLower(node.Name)) {
			
			completeNodeTask(node, commit, fmt.Sprintf("Branch '%s' pushed", branch))
		}
	}
}

// matchNodeByPrefix 根据提交消息前缀匹配节点
func matchNodeByPrefix(flowID uint, message string, commit GitCommit) {
	// 尝试提取前缀，如 "feat: xxx", "fix: xxx", "frontend: xxx"
	prefix := ""
	if idx := strings.Index(message, ":"); idx > 0 && idx < 30 {
		prefix = strings.TrimSpace(message[:idx])
	}

	if prefix == "" {
		// 使用第一个词
		parts := strings.Fields(message)
		if len(parts) > 0 {
			prefix = parts[0]
		}
	}

	var nodes []model.AgentNode
	db.DB.Where("flow_id = ?", flowID).Find(&nodes)

	for _, node := range nodes {
		nodeName := strings.ToLower(node.Name)
		if strings.Contains(nodeName, strings.ToLower(prefix)) ||
			strings.Contains(strings.ToLower(prefix), nodeName) {
			completeNodeTask(node, commit, fmt.Sprintf("Commit prefix '%s' matched", prefix))
		}
	}
}

// matchNodeByTag 根据标签匹配
func matchNodeByTag(flowID uint, message string, branch string, commit GitCommit) {
	// 检查提交消息中的标签，如 #frontend, #backend
	tags := extractTags(message)
	
	var nodes []model.AgentNode
	db.DB.Where("flow_id = ?", flowID).Find(&nodes)

	for _, node := range nodes {
		nodeName := strings.ToLower(node.Name)
		for _, tag := range tags {
			if strings.Contains(nodeName, tag) {
				completeNodeTask(node, commit, fmt.Sprintf("Tag '#%s' matched", tag))
				break
			}
		}
	}
}

// completeRunningNodes 完成所有运行中的节点任务
func completeRunningNodes(flowID uint, commit GitCommit) {
	var nodes []model.AgentNode
	db.DB.Where("flow_id = ?", flowID).Find(&nodes)

	for _, node := range nodes {
		if node.Type == "agent" {
			completeNodeTask(node, commit, "Auto-completed by push")
		}
	}
}

// completeNodeTask 完成节点关联的 Agent 任务
func completeNodeTask(node model.AgentNode, commit GitCommit, reason string) {
	taskID := fmt.Sprintf("flow_webhook_%d_%s", node.FlowID, node.NodeID)

	// 查找或创建任务
	var task model.AgentTask
	result := db.DB.Where("id = ? AND status IN ?", taskID, []string{"queued", "running"}).First(&task)
	
	if result.Error == nil {
		// 更新现有任务
		db.DB.Model(&task).Updates(map[string]interface{}{
			"status":       "completed",
			"completed_at": time.Now(),
		})
		db.DB.Create(&model.AgentTaskLog{
			TaskID:  taskID,
			Type:    "system",
			Content: fmt.Sprintf("[Webhook] %s - %s by %s: %s", reason, commit.Branch, commit.Author, commit.Message),
		})
		
		log.Printf("[Webhook] Task %s completed: %s", taskID, reason)
	} else {
		// 创建已完成的任务
		task = model.AgentTask{
			ID:          taskID,
			AgentID:     parseAgentID(node.AgentID),
			Title:       fmt.Sprintf("[Webhook] %s", node.Name),
			Prompt:      commit.Message,
			Status:      "completed",
			CompletedAt: time.Now(),
		}
		db.DB.Create(&task)
		db.DB.Create(&model.AgentTaskLog{
			TaskID:  taskID,
			Type:    "system",
			Content: fmt.Sprintf("[Webhook] %s - %s by %s", reason, commit.Branch, commit.Author),
		})
		
		log.Printf("[Webhook] Created completed task %s for node %s", taskID, node.Name)
	}

	// 更新提交记录
	commit.TaskID = taskID
	db.DB.Model(&commit).Update("task_id", taskID)
}

// extractRepoKey 从 URL 提取仓库标识
func extractRepoKey(url string) string {
	// https://github.com/user/repo -> user/repo
	// git@github.com:user/repo.git -> user/repo
	url = strings.TrimSuffix(url, ".git")
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		return url[idx+1:]
	}
	return url
}

// extractTags 从消息中提取标签
func extractTags(message string) []string {
	var tags []string
	for _, word := range strings.Fields(message) {
		if strings.HasPrefix(word, "#") && len(word) > 1 {
			tags = append(tags, strings.ToLower(strings.TrimPrefix(word, "#")))
		}
	}
	return tags
}

// parseAgentID 解析 Agent ID
func parseAgentID(s string) uint {
	var id uint
	fmt.Sscanf(s, "%d", &id)
	return id
}

// ─── Webhook 配置 API ───

// GetRepoMappings 获取仓库映射
func GetRepoMappings() []GitRepoMapping {
	var mappings []GitRepoMapping
	db.DB.Order("created_at desc").Find(&mappings)
	return mappings
}

// CreateRepoMapping 创建映射
func CreateRepoMapping(repoURL, repoName string, flowID uint, branch, nodeMatch string) *GitRepoMapping {
	m := &GitRepoMapping{
		RepoURL:   repoURL,
		RepoName:  repoName,
		FlowID:    flowID,
		Branch:    branch,
		NodeMatch: nodeMatch,
	}
	db.DB.Create(m)
	return m
}

// DeleteRepoMapping 删除映射
func DeleteRepoMapping(id uint) error {
	return db.DB.Delete(&GitRepoMapping{}, id).Error
}

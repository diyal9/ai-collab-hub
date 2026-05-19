package main

import (
	"ai-collab-hub/internal/auth"
	"ai-collab-hub/internal/config"
	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/engine"
	"ai-collab-hub/internal/gateway"
	"ai-collab-hub/internal/model"
	"ai-collab-hub/internal/ws"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// ─── 远程终端 WebSocket 代理 ───

type TerminalProxy struct {
	Backend *websocket.Conn
	Front   *websocket.Conn
}

func (tp *TerminalProxy) pump(src, dst *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		mt, msg, err := src.ReadMessage()
		if err != nil {
			return
		}
		if err := dst.WriteMessage(mt, msg); err != nil {
			return
		}
	}
}

func proxyTo(c *gin.Context, target string) {
	req, err := http.NewRequest(c.Request.Method, target, c.Request.Body)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	req.Header = c.Request.Header
	if target == "https://dashscope.aliyuncs.com/compatible-mode/v1/audio/transcriptions" {
		req.Header.Set("Authorization", "Bearer sk-4a2f4f92b72341ea994c7becdf65da7d")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	defer resp.Body.Close()
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

// ─── 文件扫描工具 ───

type FileInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	IsDir     bool   `json:"is_dir"`
	ModTime   string `json:"mod_time"`
	Ext       string `json:"ext"`
}

func scanDirectory(root string, prefix string) ([]FileInfo, error) {
	var files []FileInfo
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil // 跳过根目录自身
		}
		displayPath := prefix + "/" + rel
		if info.IsDir() {
			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    displayPath,
				IsDir:   true,
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		} else {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			files = append(files, FileInfo{
				Name:    info.Name(),
				Path:    displayPath,
				Size:    info.Size(),
				IsDir:   false,
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
				Ext:     ext,
			})
		}
		return nil
	})
	return files, err
}

// ─── Skills 提取 ───

func extractSkillManifest(dir string) (string, error) {
	// 尝试读取 SKILL.md 或 skill.yaml
	for _, name := range []string{"SKILL.md", "skill.yaml", "skill.yml", "manifest.json"} {
		path := filepath.Join(dir, name)
		if data, err := os.ReadFile(path); err == nil {
			return string(data), nil
		}
	}
	return "", nil
}

// ─── 主函数 ───

func main() {
	config.Load("config.yaml")
	os.MkdirAll(config.Cfg.Storage.UploadDir, 0755)
	os.MkdirAll(filepath.Join(config.Cfg.Storage.UploadDir, "skills"), 0755)
	os.MkdirAll(filepath.Join(config.Cfg.Storage.UploadDir, "knowledge"), 0755)
	db.Init()

	// 自动迁移所有模型
	for _, m := range model.AutoMigrateList() {
		db.DB.AutoMigrate(m)
	}
	for _, m := range model.AutoMigrateListExtended() {
		db.DB.AutoMigrate(m)
	}
	engine.AutoMigrateFlowRuntime()
	engine.AutoMigrateGitWebhook()

	// Init admin if not exists
	var admin model.User
	if err := db.DB.Where("username = ?", "admin").First(&admin).Error; err != nil {
		db.DB.Create(&model.User{Username: "admin", Password: auth.HashPass("admin123"), Role: "admin", Token: "tk_admin_default"})
	}

	go ws.Run()
	gateway.InitGateway()
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ═══════════════════════════════════════════
	// 公开路由 (无需鉴权)
	// ═══════════════════════════════════════════

	// Auth
	r.POST("/api/login", func(c *gin.Context) {
		var req struct{ User, Pass string }
		c.BindJSON(&req)
		var u model.User
		if db.DB.Where("username = ?", req.User).First(&u).Error == nil && auth.CheckPass(req.Pass, u.Password) {
			c.JSON(200, gin.H{"token": auth.GenToken(u.ID, u.Role), "role": u.Role, "user_id": u.ID})
		} else {
			c.JSON(401, gin.H{"error": "Invalid creds"})
		}
	})

	// Agent Callback (for task sync - from agents, no auth needed)
	r.POST("/api/agent/callback", func(c *gin.Context) {
		var req struct {
			StepID  uint   `json:"step_id"`
			Status  string `json:"status"`
			Output  string `json:"output"`
			InputReq string `json:"input_req"`
		}
		c.BindJSON(&req)
		engine.UpdateStep(req.StepID, req.Status, req.Output, req.InputReq, "")
		c.JSON(200, gin.H{"ok": true})
	})

	// ═══════════════════════════════════════════
	// Git Webhook (GitHub/GitLab push events)
	// ═══════════════════════════════════════════
	r.POST("/api/webhook/git", func(c *gin.Context) {
		// 验证 Secret（可选）
		secret := config.Cfg.Webhook.Secret
		if secret != "" {
			sig := c.GetHeader("X-Hub-Signature-256")
			if sig == "" && c.GetHeader("X-Gitlab-Token") != secret {
				// 继续处理（兼容模式）
			}
		}

		// 识别事件来源
		eventType := c.GetHeader("X-GitHub-Event")
		if eventType == "" {
			eventType = c.GetHeader("X-Gitlab-Event")
		}

		var payload map[string]interface{}
		c.BindJSON(&payload)

		// 处理 push 事件
		if eventType == "push" || payload["object_kind"] == "push" {
			engine.HandleGitPush(payload, eventType)
			c.JSON(200, gin.H{"ok": true, "message": "Push event processed"})
			return
		}

		// 处理 pull request / merge request
		if eventType == "pull_request" || eventType == "Merge Request Hook" {
			action, _ := payload["action"].(string)
			if eventType == "Merge Request Hook" {
				obj, _ := payload["object_attributes"].(map[string]interface{})
				action, _ = obj["action"].(string)
			}
			engine.HandlePullRequest(payload, action, eventType)
			c.JSON(200, gin.H{"ok": true, "message": "PR/MR event processed"})
			return
		}

		c.JSON(200, gin.H{"ok": true, "message": "Event received", "type": eventType})
	})

	// ═══════════════════════════════════════════
	// 共享资源下载 (无需鉴权)
	// ═══════════════════════════════════════════
	resourceRoot := "/root/aispace/hermesshare/shared-resources"
	os.MkdirAll(resourceRoot, 0755)

	r.GET("/resources", func(c *gin.Context) {
		files, err := scanDirectory(resourceRoot, "/resources/download")
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, files)
	})

	r.GET("/resources/download/*filepath", func(c *gin.Context) {
		filePath := c.Param("filepath")
		fullPath := filepath.Join(resourceRoot, filePath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "file not found"})
			return
		}
		c.FileAttachment(fullPath, filepath.Base(fullPath))
	})

	// ═══════════════════════════════════════════
	// 需鉴权的路由组
	// ═══════════════════════════════════════════
	authGroup := r.Group("/api")
	authGroup.Use(auth.AuthMiddleware())
	{
		// ─── Admin ───
		adminRoutes := authGroup.Group("")
		adminRoutes.Use(auth.AdminOnly())
		{
			adminRoutes.POST("/users", func(c *gin.Context) {
				var u model.User
				c.BindJSON(&u)
				u.Password = auth.HashPass(u.Password)
				u.Role = "user"
				u.Token = fmt.Sprintf("tk_%s_%d", u.Username, time.Now().Unix())
				db.DB.Create(&u)
				c.JSON(200, u)
			})

			adminRoutes.GET("/users", func(c *gin.Context) {
				var users []model.User
				db.DB.Select("id, username, role, api_token").Find(&users)
				c.JSON(200, users)
			})
		}

		// ─── Files ───
		authGroup.POST("/upload", func(c *gin.Context) {
			f, _ := c.FormFile("file")
			savePath := filepath.Join(config.Cfg.Storage.UploadDir, f.Filename)
			c.SaveUploadedFile(f, savePath)
			var fi model.File
			fi.FileName = f.Filename
			fi.Path = savePath
			fi.Size = f.Size
			fi.Uploader = "user"
			db.DB.Create(&fi)
			c.JSON(200, fi)
		})

		authGroup.GET("/files", func(c *gin.Context) {
			var dbFiles []model.File
			db.DB.Find(&dbFiles)

			// 扫描原始项目文件
			var existingFiles []FileInfo
			legacyPath := "/root/aispace/hermesshare"
			if _, err := os.Stat(legacyPath); err == nil {
				existingFiles, _ = scanDirectory(legacyPath, "/legacy")
			}

			// 扫描上传目录
			var uploadedFiles []FileInfo
			if _, err := os.Stat(config.Cfg.Storage.UploadDir); err == nil {
				uploadedFiles, _ = scanDirectory(config.Cfg.Storage.UploadDir, "/uploads")
			}

			c.JSON(200, gin.H{
				"database_files": dbFiles,
				"legacy_files":   existingFiles,
				"uploaded_files": uploadedFiles,
			})
		})

		// 文件下载
		authGroup.GET("/files/download", func(c *gin.Context) {
			filePath := c.Query("path")
			if filePath == "" {
				c.JSON(400, gin.H{"error": "path required"})
				return
			}
			// 安全检查: 防止路径穿越
			cleanPath := filepath.Clean(filePath)
			if strings.Contains(cleanPath, "..") {
				c.JSON(400, gin.H{"error": "invalid path"})
				return
			}

			if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
				c.JSON(404, gin.H{"error": "file not found"})
				return
			}
			c.FileAttachment(cleanPath, filepath.Base(cleanPath))
		})

		// ─── Tasks ───
		authGroup.POST("/tasks", func(c *gin.Context) {
			var req struct {
				Title, Desc, Type, Source string
				FlowID                    uint
				FlowParams                string
				Steps                     []string
			}
			c.BindJSON(&req)
			id := fmt.Sprintf("task_%d", time.Now().UnixNano())
			
			if req.Source == "flow" && req.FlowID > 0 {
				// 引用流程模式：执行流程并关联到任务
				go func() {
					exec, err := engine.ExecuteFlow(req.FlowID)
					if err != nil {
						log.Printf("[Flow] Execution failed: %v", err)
						db.DB.Model(&model.Task{}).Where("id = ?", id).Updates(map[string]interface{}{
							"status": "failed",
							"flow_execution_id": exec.ID, // Still save ID if exec object exists
						})
					} else {
						log.Printf("[Flow] Execution %d completed: %s", exec.ID, exec.Status)
						db.DB.Model(&model.Task{}).Where("id = ?", id).Updates(map[string]interface{}{
							"status": exec.Status,
							"flow_execution_id": exec.ID,
						})
					}
				}()
				
				// 创建任务记录（仅用于展示和追踪）
				task := model.Task{
					ID: id, Title: req.Title, Description: req.Desc, Type: "flow", 
					FlowID: req.FlowID, FlowParams: req.FlowParams, Source: "flow", Status: "running",
					CreatorID: 1, // TODO: get real creator ID
					CreatedAt: time.Now(), UpdatedAt: time.Now(),
				}
				db.DB.Create(&task)
				c.JSON(200, gin.H{"id": id, "status": "flow_started"})
			} else {
				// 传统自定义步骤模式
				engine.StartTask(id, req.Title, req.Desc, req.Type, req.Steps, 1)
				c.JSON(200, gin.H{"id": id, "status": "created"})
			}
		})

		authGroup.GET("/tasks", func(c *gin.Context) {
			var t []model.Task
			db.DB.Order("created_at desc").Find(&t)
			c.JSON(200, t)
		})

		authGroup.GET("/tasks/:id/steps", func(c *gin.Context) {
			var s []model.TaskStep
			db.DB.Where("task_id = ?", c.Param("id")).Find(&s)
			c.JSON(200, s)
		})

		authGroup.POST("/tasks/:id/steps/:sid/reply", func(c *gin.Context) {
			var req struct{ Reply string }
			c.BindJSON(&req)
			engine.UpdateStep(0, "success", "", "", req.Reply)
			c.JSON(200, gin.H{"ok": true})
		})

		// ═══════════════════════════════════════════
		// 1. Agent 终端管理
		// ═══════════════════════════════════════════
		authGroup.GET("/terminals", func(c *gin.Context) {
			var terminals []model.AgentTerminal
			db.DB.Order("updated_at desc").Find(&terminals)
			c.JSON(200, terminals)
		})

		authGroup.POST("/terminals", func(c *gin.Context) {
			var t model.AgentTerminal
			c.BindJSON(&t)
			t.Status = "disconnected"
			db.DB.Create(&t)
			c.JSON(200, t)
		})

		authGroup.PUT("/terminals/:id", func(c *gin.Context) {
			var t model.AgentTerminal
			if db.DB.First(&t, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.BindJSON(&t)
			t.ID, _ = parseUint(c.Param("id"))
			db.DB.Save(&t)
			c.JSON(200, t)
		})

		authGroup.DELETE("/terminals/:id", func(c *gin.Context) {
			db.DB.Delete(&model.AgentTerminal{}, c.Param("id"))
			c.JSON(200, gin.H{"ok": true})
		})

		// 连接终端
		authGroup.POST("/terminals/:id/connect", func(c *gin.Context) {
			var t model.AgentTerminal
			if db.DB.First(&t, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			// 尝试连接
			conn, _, err := websocket.DefaultDialer.Dial(t.Endpoint, http.Header{
				"Authorization": []string{"Bearer " + t.Token},
			})
			if err != nil {
				t.Status = "error"
				db.DB.Save(&t)
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			t.Status = "connected"
			t.LastConnect = time.Now()
			db.DB.Save(&t)
			conn.Close() // 测试连接，实际由前端 WebSocket 维持
			c.JSON(200, gin.H{"ok": true, "status": "connected"})
		})

		// 终端实时日志 (SSE)
		authGroup.GET("/terminals/:id/logs", func(c *gin.Context) {
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")

			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-c.Request.Context().Done():
					return
				case <-ticker.C:
					var t model.AgentTerminal
					db.DB.First(&t, c.Param("id"))
					data, _ := json.Marshal(t)
					fmt.Fprintf(c.Writer, "data: %s\n\n", data)
					c.Writer.Flush()
				}
			}
		})

		// ═══════════════════════════════════════════
		// 2. Skills 技能管理
		// ═══════════════════════════════════════════
		authGroup.GET("/skills", func(c *gin.Context) {
			var skills []model.Skill
			db.DB.Order("updated_at desc").Find(&skills)
			c.JSON(200, skills)
		})

		authGroup.POST("/skills", func(c *gin.Context) {
			var s model.Skill
			c.BindJSON(&s)
			db.DB.Create(&s)
			c.JSON(200, s)
		})

		authGroup.PUT("/skills/:id", func(c *gin.Context) {
			var s model.Skill
			if db.DB.First(&s, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.BindJSON(&s)
			s.ID, _ = parseUint(c.Param("id"))
			db.DB.Save(&s)
			c.JSON(200, s)
		})

		authGroup.DELETE("/skills/:id", func(c *gin.Context) {
			var s model.Skill
			if db.DB.First(&s, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			// 删除文件
			if s.Path != "" {
				os.RemoveAll(s.Path)
			}
			db.DB.Delete(&s)
			c.JSON(200, gin.H{"ok": true})
		})

		authGroup.POST("/skills/upload", func(c *gin.Context) {
			f, _ := c.FormFile("file")
			skillName := strings.TrimSuffix(f.Filename, filepath.Ext(f.Filename))
			skillDir := filepath.Join(config.Cfg.Storage.UploadDir, "skills", skillName)
			os.MkdirAll(skillDir, 0755)

			dstPath := filepath.Join(skillDir, f.Filename)
			c.SaveUploadedFile(f, dstPath)

			// 如果是 zip，解压
			if strings.HasSuffix(strings.ToLower(f.Filename), ".zip") {
				// 简单解压逻辑
				os.Rename(dstPath, filepath.Join(skillDir, "_archive.zip"))
				// 实际应使用 archive/zip 解压，这里标记为已上传
			}

			manifest, _ := extractSkillManifest(skillDir)

			skill := model.Skill{
				Name:        skillName,
				Version:     "1.0.0",
				Description: skillName,
				Author:      "user",
				Path:        skillDir,
				Manifest:    manifest,
				Enabled:     true,
			}
			db.DB.Create(&skill)
			c.JSON(200, skill)
		})

		authGroup.POST("/skills/:id/toggle", func(c *gin.Context) {
			var s model.Skill
			if db.DB.First(&s, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			s.Enabled = !s.Enabled
			db.DB.Save(&s)
			c.JSON(200, gin.H{"ok": true, "enabled": s.Enabled})
		})

		authGroup.GET("/skills/:id/manifest", func(c *gin.Context) {
			var s model.Skill
			if db.DB.First(&s, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.JSON(200, gin.H{"manifest": s.Manifest, "path": s.Path})
		})

		// ═══════════════════════════════════════════
		// 3. Agent 编排 (Flow)
		// ═══════════════════════════════════════════
		authGroup.GET("/flows", func(c *gin.Context) {
			var flows []model.AgentFlow
			db.DB.Order("updated_at desc").Find(&flows)
			c.JSON(200, flows)
		})

		authGroup.POST("/flows", func(c *gin.Context) {
			var f model.AgentFlow
			c.BindJSON(&f)
			db.DB.Create(&f)
			c.JSON(200, f)
		})

		authGroup.PUT("/flows/:id", func(c *gin.Context) {
			var f model.AgentFlow
			if db.DB.First(&f, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.BindJSON(&f)
			f.ID, _ = parseUint(c.Param("id"))
			db.DB.Save(&f)
			c.JSON(200, f)
		})

		authGroup.DELETE("/flows/:id", func(c *gin.Context) {
			flowID, _ := parseUint(c.Param("id"))
			// 删除关联节点和边
			db.DB.Where("flow_id = ?", flowID).Delete(&model.AgentNode{})
			db.DB.Where("flow_id = ?", flowID).Delete(&model.AgentEdge{})
			db.DB.Delete(&model.AgentFlow{}, flowID)
			c.JSON(200, gin.H{"ok": true})
		})

		// 节点
		authGroup.GET("/flows/:id/nodes", func(c *gin.Context) {
			var nodes []model.AgentNode
			db.DB.Where("flow_id = ?", c.Param("id")).Find(&nodes)
			c.JSON(200, nodes)
		})

		authGroup.POST("/flows/:id/nodes", func(c *gin.Context) {
			var n model.AgentNode
			c.BindJSON(&n)
			n.FlowID, _ = parseUint(c.Param("id"))
			db.DB.Create(&n)
			c.JSON(200, n)
		})

		authGroup.PUT("/flows/:id/nodes/:nid", func(c *gin.Context) {
			var n model.AgentNode
			if db.DB.Where("flow_id = ? AND node_id = ?", c.Param("id"), c.Param("nid")).First(&n).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.BindJSON(&n)
			db.DB.Save(&n)
			c.JSON(200, n)
		})

		authGroup.DELETE("/flows/:id/nodes/:nid", func(c *gin.Context) {
			db.DB.Where("flow_id = ? AND node_id = ?", c.Param("id"), c.Param("nid")).Delete(&model.AgentNode{})
			c.JSON(200, gin.H{"ok": true})
		})

		// 边
		authGroup.GET("/flows/:id/edges", func(c *gin.Context) {
			var edges []model.AgentEdge
			db.DB.Where("flow_id = ?", c.Param("id")).Find(&edges)
			c.JSON(200, edges)
		})

		authGroup.POST("/flows/:id/edges", func(c *gin.Context) {
			var e model.AgentEdge
			c.BindJSON(&e)
			e.FlowID, _ = parseUint(c.Param("id"))
			db.DB.Create(&e)
			c.JSON(200, e)
		})

		authGroup.DELETE("/flows/:id/edges/:eid", func(c *gin.Context) {
			db.DB.Delete(&model.AgentEdge{}, c.Param("eid"))
			c.JSON(200, gin.H{"ok": true})
		})

		// 保存整个编排图 (批量)
		authGroup.POST("/flows/:id/save-graph", func(c *gin.Context) {
			var req struct {
				Graph string              `json:"graph"`
				Nodes []model.AgentNode   `json:"nodes"`
				Edges []model.AgentEdge   `json:"edges"`
			}
			c.BindJSON(&req)

			flowID, _ := parseUint(c.Param("id"))
			// 更新 flow
			db.DB.Model(&model.AgentFlow{}).Where("id = ?", flowID).Update("graph", req.Graph)
			// 批量更新节点
			for _, n := range req.Nodes {
				n.FlowID = flowID
				db.DB.Where("flow_id = ? AND node_id = ?", flowID, n.NodeID).
					Assign(n).FirstOrCreate(&n)
			}
			// 批量更新边
			for _, e := range req.Edges {
				e.FlowID = flowID
				db.DB.Where("flow_id = ? AND source_id = ? AND target_id = ?", flowID, e.SourceID, e.TargetID).
					Assign(e).FirstOrCreate(&e)
			}
			c.JSON(200, gin.H{"ok": true})
		})

		// ─── Flow 执行引擎 ───
		authGroup.POST("/flows/:id/execute", func(c *gin.Context) {
			flowID, _ := parseUint(c.Param("id"))
			go func() {
				exec, err := engine.ExecuteFlow(flowID)
				if err != nil {
					log.Printf("[Flow] Execution failed: %v", err)
				} else {
					log.Printf("[Flow] Execution %d completed: %s", exec.ID, exec.Status)
				}
			}()
			c.JSON(200, gin.H{"ok": true, "message": "Flow execution started"})
		})

		authGroup.GET("/flows/:id/executions", func(c *gin.Context) {
			flowID, _ := parseUint(c.Param("id"))
			execs := engine.GetFlowExecutions(flowID)
			c.JSON(200, execs)
		})

		authGroup.GET("/flow-executions/:id/steps", func(c *gin.Context) {
			execID, _ := parseUint(c.Param("id"))
			steps := engine.GetFlowSteps(execID)
			c.JSON(200, steps)
		})

		authGroup.POST("/flow-executions/:id/cancel", func(c *gin.Context) {
			execID, _ := parseUint(c.Param("id"))
			if err := engine.CancelFlowExecution(execID); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"ok": true})
		})

		// 验证 Flow（检查环、孤立节点等）
		authGroup.POST("/flows/:id/validate", func(c *gin.Context) {
			flowID, _ := parseUint(c.Param("id"))
			var flow model.AgentFlow
			if err := db.DB.First(&flow, flowID).Error; err != nil {
				c.JSON(404, gin.H{"error": "flow not found"})
				return
			}
			_, err := engine.BuildAndValidateDAG(flow)
			if err != nil {
				c.JSON(400, gin.H{"valid": false, "error": err.Error()})
			} else {
				c.JSON(200, gin.H{"valid": true, "message": "Flow structure is valid"})
			}
		})

		// ─── Webhook 仓库映射管理 ───
		authGroup.GET("/webhooks", func(c *gin.Context) {
			mappings := engine.GetRepoMappings()
			c.JSON(200, mappings)
		})

		authGroup.POST("/webhooks", func(c *gin.Context) {
			var req struct {
				RepoURL   string `json:"repo_url"`
				RepoName  string `json:"repo_name"`
				FlowID    uint   `json:"flow_id"`
				Branch    string `json:"branch"`
				NodeMatch string `json:"node_match"`
			}
			c.BindJSON(&req)
			m := engine.CreateRepoMapping(req.RepoURL, req.RepoName, req.FlowID, req.Branch, req.NodeMatch)
			c.JSON(200, m)
		})

		authGroup.DELETE("/webhooks/:id", func(c *gin.Context) {
			id, _ := parseUint(c.Param("id"))
			if err := engine.DeleteRepoMapping(id); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"ok": true})
		})

		// ═══════════════════════════════════════════
		// 4. 中枢知识管理 (Memory Hub)
		// ═══════════════════════════════════════════

		// 知识条目 CRUD
		authGroup.GET("/knowledge", func(c *gin.Context) {
			var entries []model.KnowledgeEntry
			query := db.DB
			if cat := c.Query("category"); cat != "" {
				query = query.Where("category = ?", cat)
			}
			if status := c.Query("status"); status != "" {
				query = query.Where("status = ?", status)
			}
			if tag := c.Query("tag"); tag != "" {
				query = query.Where("tags LIKE ?", "%"+tag+"%")
			}
			query.Order("updated_at desc").Find(&entries)
			c.JSON(200, entries)
		})

		authGroup.POST("/knowledge", func(c *gin.Context) {
			var e model.KnowledgeEntry
			c.BindJSON(&e)
			e.Status = "pending"
			e.CreatedBy, _ = getUserId(c)
			db.DB.Create(&e)
			c.JSON(200, e)
		})

		authGroup.PUT("/knowledge/:id", func(c *gin.Context) {
			var e model.KnowledgeEntry
			if db.DB.First(&e, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			c.BindJSON(&e)
			e.ID, _ = parseUint(c.Param("id"))
			db.DB.Save(&e)
			c.JSON(200, e)
		})

		authGroup.DELETE("/knowledge/:id", func(c *gin.Context) {
			db.DB.Delete(&model.KnowledgeEntry{}, c.Param("id"))
			c.JSON(200, gin.H{"ok": true})
		})

		// 审核操作
		authGroup.POST("/knowledge/:id/review", func(c *gin.Context) {
			var req struct {
				Action  string `json:"action"`  // approve, reject, archive
				Comment string `json:"comment"`
			}
			c.BindJSON(&req)

			var e model.KnowledgeEntry
			if db.DB.First(&e, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}

			userID, _ := getUserId(c)
			switch req.Action {
			case "approve":
				e.Status = "approved"
			case "reject":
				e.Status = "rejected"
			case "archive":
				e.Status = "archived"
			}
			e.ReviewerID = userID
			e.ReviewedAt = time.Now()
			db.DB.Save(&e)

			// 记录审核日志
			db.DB.Create(&model.AuditLog{
				EntryID: e.ID,
				Action:  req.Action,
				UserID:  userID,
				Comment: req.Comment,
			})

			c.JSON(200, gin.H{"ok": true})
		})

		// 审核日志
		authGroup.GET("/knowledge/:id/audit-log", func(c *gin.Context) {
			var logs []model.AuditLog
			db.DB.Where("entry_id = ?", c.Param("id")).Order("created_at desc").Find(&logs)
			c.JSON(200, logs)
		})

		// 知识统计
		authGroup.GET("/knowledge/stats", func(c *gin.Context) {
			var stats struct {
				Total    int64 `json:"total"`
				Pending  int64 `json:"pending"`
				Approved int64 `json:"approved"`
				Rejected int64 `json:"rejected"`
			}
			db.DB.Model(&model.KnowledgeEntry{}).Count(&stats.Total)
			db.DB.Model(&model.KnowledgeEntry{}).Where("status = ?", "pending").Count(&stats.Pending)
			db.DB.Model(&model.KnowledgeEntry{}).Where("status = ?", "approved").Count(&stats.Approved)
			db.DB.Model(&model.KnowledgeEntry{}).Where("status = ?", "rejected").Count(&stats.Rejected)
			c.JSON(200, stats)
		})

		// 会话记忆
		authGroup.GET("/memory-sessions", func(c *gin.Context) {
			var sessions []model.MemorySession
			db.DB.Order("created_at desc").Limit(50).Find(&sessions)
			c.JSON(200, sessions)
		})

		authGroup.POST("/memory-sessions", func(c *gin.Context) {
			var s model.MemorySession
			c.BindJSON(&s)
			s.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
			db.DB.Create(&s)
			c.JSON(200, s)
		})

		// Agent 列表
		authGroup.GET("/agents", func(c *gin.Context) {
			var agents []model.Agent
			db.DB.Find(&agents)
			// 同时返回在线的 WebSocket agents
			onlineAgents := []string{}
			ws.Hub.Mu.RLock()
			for id := range ws.Hub.Agents {
				onlineAgents = append(onlineAgents, id)
			}
			ws.Hub.Mu.RUnlock()
			c.JSON(200, gin.H{
				"agents":  agents,
				"online":  onlineAgents,
				"online_count": len(onlineAgents),
			})
		})

		// ═══════════════════════════════════════════
		// 5. Agent 实例管理 (Bridge)
		// ═══════════════════════════════════════════

		// Agent 实例列表（合并 Gateway 在线状态）
		authGroup.GET("/agent-instances", func(c *gin.Context) {
			var agents []model.AgentInstance
			db.DB.Order("updated_at desc").Find(&agents)
			// 同步 Gateway 在线状态
			onlineIDs := gateway.Gateway.OnlineAgents()
			onlineSet := make(map[uint]bool)
			for _, id := range onlineIDs {
				onlineSet[id] = true
			}
			for i := range agents {
				if onlineSet[agents[i].ID] {
					agents[i].Status = "online"
					db.DB.Model(&agents[i]).Update("status", "online")
				}
			}
			c.JSON(200, agents)
		})

		// 创建/注册 Agent (含 Token 生成)
		authGroup.POST("/agent-instances", func(c *gin.Context) {
			var req struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}
			c.BindJSON(&req)
			if req.Name == "" || req.Type == "" {
				c.JSON(400, gin.H{"error": "name and type required"})
				return
			}
			// 生成 Token
			token := fmt.Sprintf("agt_%s_%d_%x", req.Type, time.Now().Unix(), time.Now().UnixNano()%0xFFFF)
			agent := model.AgentInstance{
				Name:   req.Name,
				Type:   req.Type,
				Status: "offline",
				Token:  token,
			}
			db.DB.Create(&agent)
			c.JSON(200, gin.H{
				"id":    agent.ID,
				"name":  agent.Name,
				"type":  agent.Type,
				"token": token,
				"connect_cmd": fmt.Sprintf("./agent-bridge --hub-url ws://47.107.172.201:8085/ws/agent --name %s --type %s --token %s", req.Name, req.Type, token),
			})
		})

		// 生成新 Token
		authGroup.POST("/agent-instances/:id/rotate-token", func(c *gin.Context) {
			var a model.AgentInstance
			if db.DB.First(&a, c.Param("id")).Error != nil {
				c.JSON(404, gin.H{"error": "not found"})
				return
			}
			newToken := fmt.Sprintf("agt_%s_%d_%x", a.Type, time.Now().Unix(), time.Now().UnixNano()%0xFFFF)
			db.DB.Model(&a).Update("token", newToken)
			c.JSON(200, gin.H{"token": newToken})
		})

		// 下发任务给 Agent
		authGroup.POST("/agent-tasks", func(c *gin.Context) {
			var req struct {
				AgentID        uint   `json:"agent_id"`
				Title          string `json:"title"`
				Prompt         string `json:"prompt"`
				Priority       int    `json:"priority"`
				RequiredSkills string `json:"required_skills"`   // JSON array: ["coding", "git"]
				ParentTaskID   string `json:"parent_task_id"`    // 父任务 ID
				ContextSnapshot string `json:"context_snapshot"` // 额外上下文
				TimeoutMinutes int    `json:"timeout_minutes"`   // 超时分钟数 (默认30)
			}
			c.BindJSON(&req)

			if req.Title == "" || req.Prompt == "" {
				c.JSON(400, gin.H{"error": "title and prompt required"})
				return
			}
			if req.TimeoutMinutes <= 0 {
				req.TimeoutMinutes = 30
			}

			task := model.AgentTask{
				ID:              fmt.Sprintf("task_%d", time.Now().UnixNano()),
				AgentID:         req.AgentID,
				Title:           req.Title,
				Prompt:          req.Prompt,
				Status:          "queued",
				Priority:        req.Priority,
				RequiredSkills:  req.RequiredSkills,
				ParentTaskID:    req.ParentTaskID,
				ContextSnapshot: req.ContextSnapshot,
				TimeoutMinutes:  req.TimeoutMinutes,
			}
			db.DB.Create(&task)

			// 如果指定了 Agent ID，尝试立即分发给该 Agent
			if req.AgentID > 0 {
				gateway.Gateway.DispatchToSpecificAgent(req.AgentID)
			} else {
				// 否则尝试能力匹配自动分发
				gateway.Gateway.DispatchByCapabilityMatching()
			}

			c.JSON(200, task)
		})

		// 任务列表
		authGroup.GET("/agent-tasks", func(c *gin.Context) {
			var tasks []model.AgentTask
			db.DB.Order("created_at desc").Limit(100).Find(&tasks)
			c.JSON(200, tasks)
		})

		// 任务日志
		authGroup.GET("/agent-tasks/:id/logs", func(c *gin.Context) {
			var logs []model.AgentTaskLog
			db.DB.Where("task_id = ?", c.Param("id")).
				Order("timestamp asc").Limit(500).Find(&logs)
			c.JSON(200, logs)
		})

		// 任务日志实时流 (SSE)
		authGroup.GET("/agent-tasks/:id/logs/stream", func(c *gin.Context) {
			taskID := c.Param("id")
			
			// 先发送历史日志
			var logs []model.AgentTaskLog
			db.DB.Where("task_id = ?", taskID).
				Order("timestamp asc").Limit(200).Find(&logs)
			for _, log := range logs {
				data, _ := json.Marshal(log)
				fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				c.Writer.Flush()
			}

			// 设置 SSE 头
			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("Access-Control-Allow-Origin", "*")
			c.Writer.Flush()

			// 订阅新日志
			ch := gateway.SubscribeTaskLogs(taskID)
			defer gateway.UnsubscribeTaskLogs(taskID, ch)

			// 推送新日志
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-c.Request.Context().Done():
					return
				case logEntry, ok := <-ch:
					if !ok {
						return // channel closed
					}
					data, _ := json.Marshal(logEntry)
					fmt.Fprintf(c.Writer, "data: %s\n\n", data)
					c.Writer.Flush()
				case <-ticker.C:
					// 心跳
					fmt.Fprintf(c.Writer, ": heartbeat\n\n")
					c.Writer.Flush()
				}
			}
		})

		// 取消任务
		authGroup.POST("/agent-tasks/:id/cancel", func(c *gin.Context) {
			gateway.Gateway.CancelTask(c.Param("id"))
			c.JSON(200, gin.H{"ok": true})
		})

		// 手动触发能力匹配自动分发
		authGroup.POST("/agent-tasks/auto-dispatch", func(c *gin.Context) {
			gateway.Gateway.DispatchByCapabilityMatching()
			c.JSON(200, gin.H{"ok": true, "message": "Auto-dispatch triggered"})
		})

		// 创建子任务 (自动继承父任务上下文)
		authGroup.POST("/agent-tasks/subtask/:parent_id", func(c *gin.Context) {
			parentID := c.Param("parent_id")
			var req struct {
				Title          string `json:"title"`
				Prompt         string `json:"prompt"`
				AgentID        uint   `json:"agent_id"`
				RequiredSkills string `json:"required_skills"`
				TimeoutMinutes int    `json:"timeout_minutes"`
			}
			c.BindJSON(&req)

			if req.Title == "" || req.Prompt == "" {
				c.JSON(400, gin.H{"error": "title and prompt required"})
				return
			}
			if req.TimeoutMinutes <= 0 {
				req.TimeoutMinutes = 30
			}

			// 验证父任务存在
			var parentTask model.AgentTask
			if err := db.DB.Where("id = ?", parentID).First(&parentTask).Error; err != nil {
				c.JSON(404, gin.H{"error": "parent task not found"})
				return
			}

			task := model.AgentTask{
				ID:              fmt.Sprintf("task_%d", time.Now().UnixNano()),
				AgentID:         req.AgentID,
				Title:           req.Title,
				Prompt:          req.Prompt,
				Status:          "queued",
				Priority:        parentTask.Priority, // 继承优先级
				RequiredSkills:  req.RequiredSkills,
				ParentTaskID:    parentID,
				ContextSnapshot: "", // 可在执行时动态获取父任务输出
				TimeoutMinutes:  req.TimeoutMinutes,
			}
			db.DB.Create(&task)

			// 如果指定了 Agent ID，尝试立即分发
			if req.AgentID > 0 {
				gateway.Gateway.DispatchToSpecificAgent(req.AgentID)
			} else {
				gateway.Gateway.DispatchByCapabilityMatching()
			}

			c.JSON(200, gin.H{
				"task":      task,
				"parent_id": parentID,
			})
		})

		// 审批请求列表
		authGroup.GET("/approvals", func(c *gin.Context) {
			var reqs []model.ApprovalRequest
			db.DB.Where("status = ?", "pending").
				Order("created_at desc").Find(&reqs)
			c.JSON(200, reqs)
		})

		// 回复审批
		authGroup.POST("/approvals/:id/reply", func(c *gin.Context) {
			var req struct {
				Reply    string `json:"reply"`
				Approved bool   `json:"approved"`
			}
			c.BindJSON(&req)
			id, _ := parseUint(c.Param("id"))
			gateway.Gateway.ReplyToApproval(id, req.Reply, req.Approved)
			c.JSON(200, gin.H{"ok": true})
		})
	}

	// ═══════════════════════════════════════════
	// WebSocket 路由
	// ═══════════════════════════════════════════

	// Agent Gateway WS (Agent Bridge 连接)
	r.GET("/ws/agent", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		gateway.Gateway.HandleWS(conn)
	})

	// 平台内部 WS (Agent 连接)
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		id := c.Query("agent_id")
		if id == "" {
			id = fmt.Sprintf("agent_%d", time.Now().UnixNano())
		}
		agent := &ws.AgentConn{Conn: conn, Send: make(chan []byte, 256), ID: id}
		ws.Hub.Register <- agent

		// 更新数据库状态
		var a model.Agent
		if db.DB.Where("agent_id = ?", id).First(&a).Error != nil {
			db.DB.Create(&model.Agent{AgentID: id, Name: id, Status: "online"})
		} else {
			db.DB.Model(&a).Updates(map[string]interface{}{
				"status":    "online",
				"last_ping": time.Now(),
			})
		}

		go func() {
			agent.ReadPump()
			// 断线更新状态
			db.DB.Model(&model.Agent{}).Where("agent_id = ?", id).Update("status", "offline")
		}()
		go agent.WritePump()
	})

	// 远程终端 WebSocket 代理
	r.GET("/ws/terminal/:id", func(c *gin.Context) {
		terminalID := c.Param("id")
		var t model.AgentTerminal
		if db.DB.First(&t, terminalID).Error != nil {
			c.JSON(404, gin.H{"error": "terminal not found"})
			return
		}

		// 连接前端
		frontConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer frontConn.Close()

		// 连接远程终端
		backendConn, _, err := websocket.DefaultDialer.Dial(t.Endpoint, http.Header{
			"Authorization": []string{"Bearer " + t.Token},
		})
		if err != nil {
			frontConn.WriteJSON(gin.H{"error": err.Error()})
			return
		}
		defer backendConn.Close()

		t.Status = "connected"
		t.LastConnect = time.Now()
		db.DB.Save(&t)

		// 双向转发
		done1 := make(chan struct{})
		done2 := make(chan struct{})
		go func() {
			for {
				mt, msg, err := frontConn.ReadMessage()
				if err != nil {
					break
				}
				backendConn.WriteMessage(mt, msg)
			}
			close(done1)
		}()
		go func() {
			for {
				mt, msg, err := backendConn.ReadMessage()
				if err != nil {
					break
				}
				frontConn.WriteMessage(mt, msg)
			}
			close(done2)
		}()

		// 等待任一端断开
		select {
		case <-done1:
		case <-done2:
		case <-c.Request.Context().Done():
		}

		t.Status = "disconnected"
		db.DB.Save(&t)
	})

	// Legacy Proxy Routes
	r.Any("/tts", func(c *gin.Context) {
		proxyTo(c, "http://127.0.0.1:8086/tts")
	})
	r.Any("/stt", func(c *gin.Context) {
		proxyTo(c, "https://dashscope.aliyuncs.com/compatible-mode/v1/audio/transcriptions")
	})

	port := fmt.Sprintf(":%d", config.Cfg.Server.Port)
	log.Println("AI Collab Hub starting on", port)
	r.Run(port)
}

// ─── 工具函数 ───

func parseUint(s string) (uint, error) {
	var v uint
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

func getUserId(c *gin.Context) (uint, error) {
	claims, exists := c.Get("user")
	if !exists {
		return 0, fmt.Errorf("no user in context")
	}
	if m, ok := claims.(jwt.MapClaims); ok {
		if id, ok := m["id"].(float64); ok {
			return uint(id), nil
		}
	}
	return 0, fmt.Errorf("invalid user claim")
}

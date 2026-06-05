package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"ai-collab-hub/internal/config"
	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/gateway"
	"ai-collab-hub/internal/model"
)

// ─── Flow 运行时模型 ───

// FlowExecution: 一次 Flow 的执行实例
type FlowExecution struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	FlowID          uint      `json:"flow_id"`
	FlowName        string    `json:"flow_name"`
	Status          string    `json:"status"` // running, completed, failed, cancelled, waiting_approval
	Variables       string    `json:"variables" gorm:"type:text"` // JSON: 运行时变量
	PendingNodeID   string    `json:"pending_node_id"` // 等待审批的节点 ID
	PendingAt       time.Time `json:"pending_at"`      // 进入等待的时间
	ApprovalTimeout int       `json:"approval_timeout"` // 审批超时（分钟），0=不超时
	CreatedAt       time.Time `json:"created_at"`
	StartedAt       time.Time `json:"started_at"`
	EndedAt         time.Time `json:"ended_at"`
	Error           string    `json:"error"`
}

// FlowStep: 执行中的节点实例
type FlowStep struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ExecID    uint      `json:"exec_id" gorm:"index"`
	NodeID    string    `json:"node_id"`
	NodeName  string    `json:"node_name"`
	NodeType  string    `json:"node_type"`
	Status    string    `json:"status"` // pending, running, completed, failed, skipped
	Output    string    `json:"output" gorm:"type:text"`
	TaskID    string    `json:"task_id"` // 关联的 Agent 任务 ID
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Error     string    `json:"error"`
}

func AutoMigrateFlowRuntime() {
	db.DB.AutoMigrate(&FlowExecution{}, &FlowStep{})
}

// ─── DAG 拓扑排序 ───

type dagNode struct {
	ID       string
	Name     string
	NodeType string
	Config   map[string]interface{}
	EdgesIn  []string
	EdgesOut []string
}

// buildDAG 从 Flow 构建有向无环图
func buildDAG(flow model.AgentFlow) (map[string]*dagNode, error) {
	// 加载节点
	var nodes []model.AgentNode
	if err := db.DB.Where("flow_id = ?", flow.ID).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("load nodes: %w", err)
	}

	// 加载边
	var edges []model.AgentEdge
	if err := db.DB.Where("flow_id = ?", flow.ID).Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("load edges: %w", err)
	}

	graph := make(map[string]*dagNode)

	// 解析节点配置
	for _, n := range nodes {
		var cfg map[string]interface{}
		if n.Config != "" {
			json.Unmarshal([]byte(n.Config), &cfg)
		}
		graph[n.NodeID] = &dagNode{
			ID:       n.NodeID,
			Name:     n.Name,
			NodeType: n.Type,
			Config:   cfg,
		}
	}

	// 建立边关系
	for _, e := range edges {
		if src, ok := graph[e.SourceID]; ok {
			src.EdgesOut = append(src.EdgesOut, e.TargetID)
		}
		if tgt, ok := graph[e.TargetID]; ok {
			tgt.EdgesIn = append(tgt.EdgesIn, e.SourceID)
		}
	}

	// 环检测 + 拓扑排序
	if hasCycle(graph) {
		return nil, fmt.Errorf("flow contains cycle")
	}

	return graph, nil
}

// hasCycle 检测有向图是否有环 (DFS)
func hasCycle(graph map[string]*dagNode) bool {
	visited := make(map[string]int) // 0=unvisited, 1=visiting, 2=visited

	var dfs func(id string) bool
	dfs = func(id string) bool {
		if visited[id] == 1 {
			return true // 环
		}
		if visited[id] == 2 {
			return false
		}
		visited[id] = 1
		if node, ok := graph[id]; ok {
			for _, next := range node.EdgesOut {
				if dfs(next) {
					return true
				}
			}
		}
		visited[id] = 2
		return false
	}

	for id := range graph {
		if dfs(id) {
			return true
		}
	}
	return false
}

// topologicalSort 返回拓扑排序后的节点列表
func topologicalSort(graph map[string]*dagNode) []string {
	inDegree := make(map[string]int)
	for id, node := range graph {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, out := range node.EdgesOut {
			inDegree[out]++
		}
	}

	// Kahn 算法 (带闭环检测与空指针保护)
	queue := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue) // 稳定排序

	result := []string{}
	totalNodes := len(graph)
	
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		// 安全访问图节点
		if n, exists := graph[nodeID]; exists && n != nil {
			for _, next := range n.EdgesOut {
				if _, ok := inDegree[next]; ok {
					inDegree[next]--
					if inDegree[next] == 0 {
						queue = append(queue, next)
					}
				}
			}
		}
	}

	// 检测闭环：如果排序出的节点数少于总节点数，说明存在环
	if len(result) < totalNodes {
		return nil // 返回空切片表示检测到闭环
	}

	return result
}

// ─── 流程执行引擎 ───

type FlowExecutor struct {
	exec      *FlowExecution
	graph     map[string]*dagNode
	order     []string
	variables map[string]interface{}
	steps     map[string]*FlowStep
}

// ExecuteFlow 执行一个 Flow
func ExecuteFlow(flowID uint) (*FlowExecution, error) {
	var flow model.AgentFlow
	if err := db.DB.First(&flow, flowID).Error; err != nil {
		return nil, fmt.Errorf("flow not found: %w", err)
	}

	graph, err := buildDAG(flow)
	if err != nil {
		return nil, fmt.Errorf("build DAG: %w", err)
	}

	order := topologicalSort(graph)
	if len(order) == 0 {
		return nil, fmt.Errorf("no nodes in flow")
	}

	exec := &FlowExecution{
		FlowID:    flow.ID,
		FlowName:  flow.Name,
		Status:    "running",
		Variables: "{}",
		StartedAt: time.Now(),
	}
	db.DB.Create(exec)

	executor := &FlowExecutor{
		exec:      exec,
		graph:     graph,
		order:     order,
		variables: make(map[string]interface{}),
		steps:     make(map[string]*FlowStep),
	}

	log.Printf("[Flow] Executing %s (%d nodes)", flow.Name, len(order))

	// 按拓扑顺序执行
	for _, nodeID := range order {
		step, err := executor.executeNode(nodeID)
		if err != nil {
			// 审批等待：暂停流程，不标记为失败
			if IsApprovalPending(err) {
				log.Printf("[Flow] %s paused for approval at node %s", flow.Name, nodeID)
				return exec, nil
			}
			exec.Status = "failed"
			exec.Error = err.Error()
			exec.EndedAt = time.Now()
			db.DB.Save(exec)
			return exec, err
		}

		// 如果节点返回 skip，标记下游为 skipped
		if step.Status == "skipped" {
			executor.markDownstreamSkipped(nodeID)
		}

		// 如果是条件节点且结果为 false，跳过对应分支
		if step.Status == "completed" {
			if node, ok := graph[nodeID]; ok && node.NodeType == "condition" {
				result := step.Output
				if result == "false" {
					executor.skipFalseBranch(nodeID)
				}
			}
		}
	}

	exec.Status = "completed"
	exec.EndedAt = time.Now()
	db.DB.Save(exec)

	log.Printf("[Flow] %s completed", flow.Name)
	return exec, nil
}

func (e *FlowExecutor) executeNode(nodeID string) (*FlowStep, error) {
	node := e.graph[nodeID]

	// 检查前置依赖是否都完成
	for _, dep := range node.EdgesIn {
		if step, ok := e.steps[dep]; ok && (step.Status == "skipped" || step.Status == "failed") {
			// 创建 skipped 步骤
			step := &FlowStep{
				ExecID:    e.exec.ID,
				NodeID:    nodeID,
				NodeName:  node.Name,
				NodeType:  node.NodeType,
				Status:    "skipped",
				Output:    "Skipped due to upstream failure",
			}
			db.DB.Create(step)
			e.steps[nodeID] = step
			return step, nil
		}
	}

	// 创建运行中步骤
	step := &FlowStep{
		ExecID:    e.exec.ID,
		NodeID:    nodeID,
		NodeName:  node.Name,
		NodeType:  node.NodeType,
		Status:    "running",
		StartedAt: time.Now(),
	}
	db.DB.Create(step)
	e.steps[nodeID] = step

	log.Printf("[Flow] Executing node: %s (%s)", node.Name, node.NodeType)

	var err error
	switch node.NodeType {
	case "agent":
		err = e.executeAgentNode(node, step)
	case "condition":
		err = e.executeConditionNode(node, step)
	case "merge":
		err = e.executeMergeNode(node, step)
	case "trigger":
		err = e.executeTriggerNode(node, step)
	case "code":
		err = e.executeCodeNode(node, step)
	case "sandbox":
		err = e.executeSandboxNode(node, step)
	case "webhook":
		err = e.executeWebhookNode(node, step)
	case "approval":
		err = e.executeApprovalNode(node, step)
	default:
		err = fmt.Errorf("unknown node type: %s", node.NodeType)
	}

	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		step.EndedAt = time.Now()
		db.DB.Save(step)
		return step, err
	}

	step.EndedAt = time.Now()
	db.DB.Save(step)
	return step, nil
}

func (e *FlowExecutor) executeAgentNode(node *dagNode, step *FlowStep) error {
	agentIDStr, _ := node.Config["agent_id"].(string)
	prompt, _ := node.Config["prompt"].(string)
	requiredSkills, _ := node.Config["required_skills"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)

	// 模板变量替换
	prompt = e.resolveTemplate(prompt)

	// 创建 Agent 任务
	var agentID uint
	fmt.Sscanf(agentIDStr, "%d", &agentID)

	taskID := fmt.Sprintf("flow_%d_node_%s", e.exec.ID, node.ID)
	task := model.AgentTask{
		ID:             taskID,
		AgentID:        agentID,
		Title:          node.Name,
		Prompt:         prompt,
		Status:         "queued",
		Priority:       1,
		RequiredSkills: requiredSkills,
		TimeoutMinutes: int(timeoutMinutes),
	}
	if task.TimeoutMinutes <= 0 {
		task.TimeoutMinutes = 30
	}
	db.DB.Create(&task)

	step.TaskID = taskID
	db.DB.Save(step)

	// 分发任务
	if agentID > 0 {
		gateway.Gateway.DispatchToSpecificAgent(agentID)
	} else {
		gateway.Gateway.DispatchByCapabilityMatching()
	}

	// 等待任务完成（带超时）
	timeout := time.Duration(task.TimeoutMinutes) * time.Minute
	if timeout == 0 {
		timeout = 30 * time.Minute
	}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		var t model.AgentTask
		db.DB.First(&t, "id = ?", taskID)

		switch t.Status {
		case "completed":
			// 获取最终输出
			var lastLog model.AgentTaskLog
			db.DB.Where("task_id = ?", taskID).Order("timestamp desc").First(&lastLog)
			step.Status = "completed"
			step.Output = lastLog.Content
			db.DB.Save(step)

			// 存储到变量
			e.variables[node.ID] = lastLog.Content
			e.variables[node.ID+"_task_id"] = taskID
			e.saveVariables()
			return nil

		case "failed":
			step.Status = "failed"
			step.Error = t.Error
			db.DB.Save(step)
			return fmt.Errorf("agent task failed: %s", t.Error)

		case "cancelled":
			step.Status = "skipped"
			step.Output = "Task cancelled"
			db.DB.Save(step)
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	step.Status = "failed"
	step.Error = "Task execution timeout"
	db.DB.Save(step)
	return fmt.Errorf("task %s timed out", taskID)
}

func (e *FlowExecutor) executeConditionNode(node *dagNode, step *FlowStep) error {
	expression, _ := node.Config["expression"].(string)

	// 解析表达式（简单实现：支持 variable == value 格式）
	expression = e.resolveTemplate(expression)

	result := e.evaluateCondition(expression)

	step.Status = "completed"
	step.Output = fmt.Sprintf("%t", result)
	db.DB.Save(step)

	e.variables[node.ID] = result
	e.saveVariables()

	log.Printf("[Flow] Condition '%s': %s = %t", node.Name, expression, result)
	return nil
}

func (e *FlowExecutor) evaluateCondition(expr string) bool {
	expr = strings.TrimSpace(expr)

	// 支持简单比较
	if strings.Contains(expr, "==") {
		parts := strings.SplitN(expr, "==", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		return left == right
	}
	if strings.Contains(expr, "!=") {
		parts := strings.SplitN(expr, "!=", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		return left != right
	}
	if strings.Contains(expr, ">") {
		parts := strings.SplitN(expr, ">", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		return left > right
	}
	if strings.Contains(expr, "<") {
		parts := strings.SplitN(expr, "<", 2)
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		return left < right
	}

	// 布尔值
	switch strings.ToLower(expr) {
	case "true", "yes", "1":
		return true
	case "false", "no", "0", "":
		return false
	}

	return true // 默认通过
}

func (e *FlowExecutor) executeMergeNode(node *dagNode, step *FlowStep) error {
	// 汇聚节点：等待所有上游完成
	var outputs []string
	for _, dep := range node.EdgesIn {
		if s, ok := e.steps[dep]; ok {
			outputs = append(outputs, fmt.Sprintf("[%s]: %s", s.NodeName, s.Output))
		}
	}

	step.Status = "completed"
	step.Output = strings.Join(outputs, "\n---\n")
	db.DB.Save(step)

	e.variables[node.ID] = step.Output
	e.saveVariables()

	return nil
}

func (e *FlowExecutor) executeTriggerNode(node *dagNode, step *FlowStep) error {
	// Trigger 节点：流程起点，直接完成
	step.Status = "completed"
	step.Output = "Trigger activated"
	db.DB.Save(step)
	return nil
}

// markDownstreamSkipped 标记下游节点为 skipped
func (e *FlowExecutor) markDownstreamSkipped(nodeID string) {
	node := e.graph[nodeID]
	for _, next := range node.EdgesOut {
		if _, done := e.steps[next]; !done {
			step := &FlowStep{
				ExecID:    e.exec.ID,
				NodeID:    next,
				NodeName:  e.graph[next].Name,
				NodeType:  e.graph[next].NodeType,
				Status:    "skipped",
				Output:    "Skipped due to upstream",
			}
			db.DB.Create(step)
			e.steps[next] = step
			e.markDownstreamSkipped(next)
		}
	}
}

// skipFalseBranch 条件为 false 时，跳过不匹配的分支
func (e *FlowExecutor) skipFalseBranch(condNodeID string) {
	node := e.graph[condNodeID]
	for range node.EdgesOut {
		// TODO: 根据 edge condition 判断是否跳过
		// 简化：跳过所有出边（实际应根据边的 condition 字段）
	}
}

// resolveTemplate 解析模板变量 ${node_id}
func (e *FlowExecutor) resolveTemplate(s string) string {
	for varName, varValue := range e.variables {
		placeholder := "${" + varName + "}"
		if strings.Contains(s, placeholder) {
			// JSON-escape the value: marshal to get proper escaping, then strip surrounding quotes
			escaped, _ := json.Marshal(fmt.Sprintf("%v", varValue))
			// escaped is like `"hello\nworld"` with quotes; strip them for in-string substitution
			raw := string(escaped)
			if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
				raw = raw[1 : len(raw)-1]
			}
			s = strings.ReplaceAll(s, placeholder, raw)
		}
	}
	return s
}

func (e *FlowExecutor) saveVariables() {
	data, _ := json.Marshal(e.variables)
	e.exec.Variables = string(data)
	db.DB.Model(e.exec).Update("variables", string(data))
}

// ─── 公开 API ───

// BuildAndValidateDAG 构建并验证 DAG（公开方法供 API 调用）
func BuildAndValidateDAG(flow model.AgentFlow) (map[string]*dagNode, error) {
	graph, err := buildDAG(flow)
	if err != nil {
		return nil, err
	}
	if hasCycle(graph) {
		return nil, fmt.Errorf("flow contains cycle")
	}
	return graph, nil
}

// GetFlowExecutions 获取 Flow 执行历史
func GetFlowExecutions(flowID uint) []FlowExecution {
	var execs []FlowExecution
	query := db.DB.Order("created_at desc").Limit(50)
	if flowID > 0 {
		query = query.Where("flow_id = ?", flowID)
	}
	query.Find(&execs)
	return execs
}

// GetFlowSteps 获取执行步骤
func GetFlowSteps(execID uint) []FlowStep {
	var steps []FlowStep
	db.DB.Where("exec_id = ?", execID).Order("id asc").Find(&steps)
	return steps
}

// CancelFlowExecution 取消执行
func CancelFlowExecution(execID uint) error {
	var exec FlowExecution
	if err := db.DB.First(&exec, execID).Error; err != nil {
		return err
	}

	db.DB.Model(&exec).Updates(map[string]interface{}{
		"status":  "cancelled",
		"ended_at": time.Now(),
	})

	// 取消关联的 Agent 任务
	var steps []FlowStep
	db.DB.Where("exec_id = ? AND status = ?", execID, "running").Find(&steps)
	for _, s := range steps {
		if s.TaskID != "" {
			gateway.Gateway.CancelTask(s.TaskID)
		}
	}

	return nil
}

// executeCodeNode 执行代码节点 (shell 命令)
func (e *FlowExecutor) executeCodeNode(node *dagNode, step *FlowStep) error {
	language, _ := node.Config["language"].(string)
	code, _ := node.Config["code"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)
	
	if code == "" {
		return fmt.Errorf("code node requires 'code' config")
	}
	
	log.Printf("[Flow] Executing code node: %s (language: %s)", node.Name, language)
	
	var cmd *exec.Cmd
	timeout := time.Duration(timeoutMinutes) * time.Minute
	if timeout == 0 {
		timeout = 30 * time.Minute
	}
	
	switch strings.ToLower(language) {
	case "shell", "bash", "sh":
		cmd = exec.Command("bash", "-c", code)
	case "python", "py":
		cmd = exec.Command("python3", "-c", code)
	case "node", "js":
		cmd = exec.Command("node", "-e", code)
	default:
		cmd = exec.Command("bash", "-c", code)
	}
	
	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		step.Output = string(output)
		done <- err
	}()
	
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("code execution failed: %v, output: %s", err, step.Output)
		}
		step.Status = "completed"
		e.variables[node.ID] = step.Output
		e.saveVariables()
		return nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("code execution timed out after %v", timeout)
	}
}

// executeSandboxNode 通过 Bridge 调度 Sandbox 执行任务 (同步阻塞)
func (e *FlowExecutor) executeSandboxNode(node *dagNode, step *FlowStep) error {
	sandboxID, _ := node.Config["sandbox_id"].(string)
	executor, _ := node.Config["executor"].(string)
	command, _ := node.Config["command"].(string)
	workDir, _ := node.Config["work_dir"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)

	// Git 集成
	gitURL, _ := node.Config["git_url"].(string)
	gitBranch, _ := node.Config["git_branch"].(string)

	// Prompt 模板引用（如 "prompt:go-backend-dev-v2"）
	if strings.HasPrefix(command, "prompt:") {
		tmplName := strings.TrimPrefix(command, "prompt:")
		variables := make(map[string]string)
		if gitURL != "" {
			variables["repo_url"] = gitURL
			variables["repo_name"] = strings.TrimSuffix(strings.Split(gitURL, "/")[len(strings.Split(gitURL, "/"))-1], ".git")
		}
		if gitBranch != "" {
			variables["branch"] = gitBranch
		}
		// 注入前置节点输出为变量
		for k, v := range e.variables {
			variables[k] = fmt.Sprintf("%v", v)
		}
		resolved, err := ResolvePromptTemplate(tmplName, variables)
		if err != nil {
			return fmt.Errorf("resolve prompt template '%s': %w", tmplName, err)
		}
		command = resolved
	}

	// 模板变量替换
	command = e.resolveTemplate(command)

	if command == "" {
		return fmt.Errorf("sandbox node requires 'command' config")
	}
	if executor == "" {
		executor = "shell"
	}

	// 如果有 git_url，在命令前加 clone 步骤
	if gitURL != "" {
		if gitBranch == "" {
			gitBranch = "main"
		}
		cloneDir := workDir
		if cloneDir == "" || cloneDir == "/tmp" {
			cloneDir = "/tmp/agent-workspace"
		}
		command = fmt.Sprintf("mkdir -p %s && rm -rf %s && git clone -b %s %s %s && cd %s && %s",
			cloneDir, cloneDir, gitBranch, gitURL, cloneDir, cloneDir, command)
	}

	bridgeURL := "http://localhost:8088/api/dispatch" // Bridge HTTP 地址

	log.Printf("[Flow] Dispatching sandbox task: %s (executor=%s, sandbox=%s, git=%s)", node.Name, executor, sandboxID, gitURL)

	// 构造请求体
	payload := map[string]interface{}{
		"sandbox_id": sandboxID,
		"task_id":    fmt.Sprintf("flow_%d_node_%s", e.exec.ID, node.ID),
		"payload": map[string]interface{}{
			"executor": executor,
			"command":  command,
			"work_dir": workDir,
			"timeout":  int(timeoutMinutes) * 60,
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	resp, err := http.Post(bridgeURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to call bridge API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bridge API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		Output string `json:"output,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode bridge response: %w", err)
	}

	step.Output = result.Output
	if result.Status == "failed" {
		return fmt.Errorf("sandbox task failed: %s", result.Error)
	}

	step.Status = "completed"
	e.variables[node.ID] = result.Output
	e.saveVariables()
	return nil
}

// executeWebhookNode 执行 Webhook 节点 (HTTP 请求)
func (e *FlowExecutor) executeWebhookNode(node *dagNode, step *FlowStep) error {
	method, _ := node.Config["method"].(string)
	url, _ := node.Config["url"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)
	headersRaw, _ := node.Config["headers"].(string)
	bodyRaw, _ := node.Config["body"].(string)
	
	if url == "" {
		return fmt.Errorf("webhook node requires 'url' config")
	}
	
	url = e.resolveTemplate(url)
	
	if method == "" {
		method = "GET"
	}
	
	timeout := time.Duration(timeoutMinutes) * time.Minute
	if timeout == 0 {
		timeout = 10 * time.Minute
	}
	
	log.Printf("[Flow] Executing webhook: %s %s", method, url)
	
	// 构造 Body
	var bodyReader io.Reader
	if bodyRaw != "" {
		log.Printf("[Flow] Webhook body BEFORE template: %s", bodyRaw)
		bodyRaw = e.resolveTemplate(bodyRaw)
		log.Printf("[Flow] Webhook body AFTER template: %s", bodyRaw)
		bodyReader = strings.NewReader(bodyRaw)
	}
	
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// 设置 Headers
	if headersRaw != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(headersRaw), &headers); err == nil {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
		}
	}
	
	// 默认 Content-Type
	if req.Header.Get("Content-Type") == "" && bodyRaw != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	log.Printf("[Flow] Webhook response: status=%d, body=%s", resp.StatusCode, string(body))
	step.Output = fmt.Sprintf("Status: %d\nBody: %s", resp.StatusCode, string(body))
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d, body: %s", resp.StatusCode, string(body))
	}
	
	step.Status = "completed"
	e.variables[node.ID] = step.Output
	e.saveVariables()
	return nil
}

// executeApprovalNode 人在回路：发送飞书审批卡片并暂停流程
func (e *FlowExecutor) executeApprovalNode(node *dagNode, step *FlowStep) error {
	webhookURL, _ := node.Config["webhook_url"].(string)
	cardTitle, _ := node.Config["card_title"].(string)
	cardBody, _ := node.Config["card_body"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)
	_ = node.Config["on_timeout"] // onTimeout: 预留字段

	if webhookURL == "" {
		return fmt.Errorf("approval node requires 'webhook_url' config")
	}
	if cardTitle == "" {
		cardTitle = "🔧 AI 任务审批"
	}

	// 模板变量替换
	cardBody = e.resolveTemplate(cardBody)
	cardTitle = e.resolveTemplate(cardTitle)

	// 构造飞书审批卡片
	card := map[string]interface{}{
		"config": map[string]interface{}{"wide_screen_mode": true},
		"header": map[string]interface{}{
			"title":    map[string]interface{}{"tag": "plain_text", "content": cardTitle},
			"template": "blue",
		},
		"elements": []interface{}{
			map[string]interface{}{
				"tag":     "markdown",
				"content": cardBody,
			},
			map[string]interface{}{
				"tag": "action",
				"actions": []interface{}{
					map[string]interface{}{
						"tag":  "button",
						"text": map[string]interface{}{"tag": "plain_text", "content": "✅ 批准执行"},
						"type": "primary",
						"url":  fmt.Sprintf("%s/api/flows/%d/executions/%d/approve?token=%s", getPublicURL(), e.exec.FlowID, e.exec.ID, generateApprovalToken(e.exec.ID)),
					},
					map[string]interface{}{
						"tag":  "button",
						"text": map[string]interface{}{"tag": "plain_text", "content": "❌ 拒绝"},
						"type": "danger",
						"url":  fmt.Sprintf("%s/api/flows/%d/executions/%d/reject?token=%s", getPublicURL(), e.exec.FlowID, e.exec.ID, generateApprovalToken(e.exec.ID)),
					},
				},
			},
		},
	}

	bodyPayload := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}
	bodyBytes, _ := json.Marshal(bodyPayload)

	log.Printf("[Flow] Sending approval card to %s", webhookURL)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to send approval card: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("approval card send failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	// 标记流程为等待审批状态
	step.Status = "running" // 保持 running，等待回调
	step.Output = "Waiting for approval. Card sent to Feishu."
	step.EndedAt = time.Now()
	db.DB.Save(step)

	e.exec.Status = "waiting_approval"
	e.exec.PendingNodeID = node.ID
	e.exec.PendingAt = time.Now()
	e.exec.ApprovalTimeout = int(timeoutMinutes)
	if e.exec.ApprovalTimeout == 0 {
		e.exec.ApprovalTimeout = 1440 // 默认 24 小时
	}
	db.DB.Save(e.exec)

	// 返回特殊 error 让执行器暂停（不标记为 failed）
	return &ApprovalPendingError{ExecID: e.exec.ID}
}

// ApprovalPendingError 表示审批等待中（非真正错误）
type ApprovalPendingError struct {
	ExecID uint
}

func (e *ApprovalPendingError) Error() string {
	return fmt.Sprintf("approval pending for execution %d", e.ExecID)
}

// IsApprovalPending 判断是否为审批等待错误
func IsApprovalPending(err error) bool {
	_, ok := err.(*ApprovalPendingError)
	return ok
}

// ResumeFlowExecution 恢复被暂停的审批流程
func ResumeFlowExecution(execID uint) error {
	var exec FlowExecution
	if err := db.DB.First(&exec, execID).Error; err != nil {
		return fmt.Errorf("execution not found: %w", err)
	}
	if exec.Status != "waiting_approval" {
		return fmt.Errorf("execution is not waiting for approval (current: %s)", exec.Status)
	}

	log.Printf("[Flow] Resuming execution %d (approved)", execID)

	// 更新执行状态
	exec.Status = "running"
	exec.PendingNodeID = ""
	exec.PendingAt = time.Time{}
	db.DB.Save(&exec)

	// 重新加载图并继续执行
	var flow model.AgentFlow
	if err := db.DB.First(&flow, exec.FlowID).Error; err != nil {
		return fmt.Errorf("flow not found: %w", err)
	}

	graph, err := buildDAG(flow)
	if err != nil {
		return fmt.Errorf("build DAG: %w", err)
	}

	// 恢复变量
	variables := make(map[string]interface{})
	if exec.Variables != "" {
		json.Unmarshal([]byte(exec.Variables), &variables)
	}

	// 从审批节点之后继续
	pendingNodeID := exec.PendingNodeID
	// 标记审批节点为完成
	var step FlowStep
	if err := db.DB.Where("exec_id = ? AND node_id = ?", execID, pendingNodeID).First(&step).Error; err == nil {
		step.Status = "completed"
		step.Output = "Approved by user."
		step.EndedAt = time.Now()
		db.DB.Save(&step)
		variables[pendingNodeID] = "approved"
	}

	// 重建执行器继续
	order := topologicalSort(graph)
	executor := &FlowExecutor{
		exec:      &exec,
		graph:     graph,
		order:     order,
		variables: variables,
		steps:     make(map[string]*FlowStep),
	}
	// 加载已有步骤
	var steps []FlowStep
	db.DB.Where("exec_id = ?", execID).Find(&steps)
	for i := range steps {
		executor.steps[steps[i].NodeID] = &steps[i]
	}

	// 从审批节点之后继续执行
	skip := true
	for _, nodeID := range order {
		if nodeID == pendingNodeID {
			skip = false
			continue
		}
		if skip {
			continue
		}
		_, err := executor.executeNode(nodeID)
		if err != nil {
			exec.Status = "failed"
			exec.Error = err.Error()
			exec.EndedAt = time.Now()
			db.DB.Save(&exec)
			return err
		}
	}

	exec.Status = "completed"
	exec.EndedAt = time.Now()
	db.DB.Save(&exec)

	log.Printf("[Flow] Execution %d completed after approval", execID)
	return nil
}

// AbortFlowExecution 终止被暂停的审批流程
func AbortFlowExecution(execID uint) error {
	var exec FlowExecution
	if err := db.DB.First(&exec, execID).Error; err != nil {
		return fmt.Errorf("execution not found: %w", err)
	}
	if exec.Status != "waiting_approval" {
		return fmt.Errorf("execution is not waiting for approval (current: %s)", exec.Status)
	}

	log.Printf("[Flow] Aborting execution %d (rejected)", execID)

	// 标记审批节点为失败
	var step FlowStep
	if exec.PendingNodeID != "" {
		db.DB.Where("exec_id = ? AND node_id = ?", exec.ID, exec.PendingNodeID).First(&step)
		if step.ID > 0 {
			step.Status = "failed"
			step.Output = "Rejected by user."
			step.Error = "User rejected approval"
			step.EndedAt = time.Now()
			db.DB.Save(&step)
		}
	}

	// 标记所有 running 步骤为 skipped
	db.DB.Model(&FlowStep{}).Where("exec_id = ? AND status = ?", execID, "running").Updates(map[string]interface{}{
		"status": "skipped",
		"output": "Skipped: approval rejected",
	})

	exec.Status = "failed"
	exec.Error = "Approval rejected by user"
	exec.PendingNodeID = ""
	exec.EndedAt = time.Now()
	db.DB.Save(&exec)

	return nil
}

// ResolvePromptTemplate 获取提示词模板（Langfuse 优先，本地 DB 后备）
func ResolvePromptTemplate(name string, variables map[string]string) (string, error) {
	// 1. 尝试从 Langfuse 获取
	if config.Cfg.Langfuse.URL != "" && config.Cfg.Langfuse.PublicKey != "" && config.Cfg.Langfuse.SecretKey != "" {
		content, err := fetchLangfusePrompt(name, variables)
		if err == nil {
			return content, nil
		}
		log.Printf("[Prompt] Langfuse fetch failed for %s: %v, falling back to local DB", name, err)
	}

	// 2. 本地 DB 后备
	var tmpl model.PromptTemplate
	if err := db.DB.Where("name = ?", name).Order("version desc").First(&tmpl).Error; err != nil {
		return "", fmt.Errorf("prompt template '%s' not found in local DB", name)
	}

	content := tmpl.Content
	for k, v := range variables {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content, nil
}

// fetchLangfusePrompt 从 Langfuse 获取提示词
func fetchLangfusePrompt(name string, variables map[string]string) (string, error) {
	url := strings.TrimRight(config.Cfg.Langfuse.URL, "/") + "/api/public/prompts/" + name
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(config.Cfg.Langfuse.PublicKey, config.Cfg.Langfuse.SecretKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("langfuse API error: %d %s", resp.StatusCode, string(body))
	}

	var result struct {
		Prompt string `json:"prompt"`
		Labels []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	content := result.Prompt
	for k, v := range variables {
		content = strings.ReplaceAll(content, "{{"+k+"}}", v)
	}
	return content, nil
}

// getPublicURL 获取 Hub 公网地址（用于审批回调）
func getPublicURL() string {
	if config.Cfg.Server.PublicURL != "" {
		return config.Cfg.Server.PublicURL
	}
	return "http://localhost:8087"
}

// generateApprovalToken 生成审批回调 token（简单实现，生产环境应使用 JWT）
func generateApprovalToken(execID uint) string {
	return fmt.Sprintf("approval_%d_%d", execID, time.Now().Unix())
}

// SavePromptTemplate 保存提示词模板
func SavePromptTemplate(tmpl *model.PromptTemplate) error {
	return db.DB.Save(tmpl).Error
}

// GetPromptTemplates 获取所有提示词模板
func GetPromptTemplates() []model.PromptTemplate {
	var templates []model.PromptTemplate
	db.DB.Order("name asc, version desc").Find(&templates)
	return templates
}

// DeletePromptTemplate 删除提示词模板
func DeletePromptTemplate(id uint) error {
	return db.DB.Delete(&model.PromptTemplate{}, id).Error
}

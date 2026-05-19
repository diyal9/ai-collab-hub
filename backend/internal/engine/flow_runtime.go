package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"time"

	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/gateway"
	"ai-collab-hub/internal/model"
)

// ─── Flow 运行时模型 ───

// FlowExecution: 一次 Flow 的执行实例
type FlowExecution struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	FlowID    uint      `json:"flow_id"`
	FlowName  string    `json:"flow_name"`
	Status    string    `json:"status"` // running, completed, failed, cancelled, waiting
	Variables string    `json:"variables" gorm:"type:text"` // JSON: 运行时变量
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Error     string    `json:"error"`
	CreatedAt time.Time `json:"created_at"`
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
	id       string
	name     string
	nodeType string
	config   map[string]interface{}
	edgesIn  []string
	edgesOut []string
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
			id:       n.NodeID,
			name:     n.Name,
			nodeType: n.Type,
			config:   cfg,
		}
	}

	// 建立边关系
	for _, e := range edges {
		if src, ok := graph[e.SourceID]; ok {
			src.edgesOut = append(src.edgesOut, e.TargetID)
		}
		if tgt, ok := graph[e.TargetID]; ok {
			tgt.edgesIn = append(tgt.edgesIn, e.SourceID)
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
			for _, next := range node.edgesOut {
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
		for _, out := range node.edgesOut {
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
			for _, next := range n.edgesOut {
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
			if node, ok := graph[nodeID]; ok && node.nodeType == "condition" {
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
	for _, dep := range node.edgesIn {
		if step, ok := e.steps[dep]; ok && (step.Status == "skipped" || step.Status == "failed") {
			// 创建 skipped 步骤
			step := &FlowStep{
				ExecID:    e.exec.ID,
				NodeID:    nodeID,
				NodeName:  node.name,
				NodeType:  node.nodeType,
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
		NodeName:  node.name,
		NodeType:  node.nodeType,
		Status:    "running",
		StartedAt: time.Now(),
	}
	db.DB.Create(step)
	e.steps[nodeID] = step

	log.Printf("[Flow] Executing node: %s (%s)", node.name, node.nodeType)

	var err error
	switch node.nodeType {
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
	case "webhook":
		err = e.executeWebhookNode(node, step)
	default:
		err = fmt.Errorf("unknown node type: %s", node.nodeType)
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
	agentIDStr, _ := node.config["agent_id"].(string)
	prompt, _ := node.config["prompt"].(string)
	requiredSkills, _ := node.config["required_skills"].(string)
	timeoutMinutes, _ := node.config["timeout_minutes"].(float64)

	// 模板变量替换
	prompt = e.resolveTemplate(prompt)

	// 创建 Agent 任务
	var agentID uint
	fmt.Sscanf(agentIDStr, "%d", &agentID)

	taskID := fmt.Sprintf("flow_%d_node_%s", e.exec.ID, node.id)
	task := model.AgentTask{
		ID:             taskID,
		AgentID:        agentID,
		Title:          node.name,
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
			e.variables[node.id] = lastLog.Content
			e.variables[node.id+"_task_id"] = taskID
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
	expression, _ := node.config["expression"].(string)

	// 解析表达式（简单实现：支持 variable == value 格式）
	expression = e.resolveTemplate(expression)

	result := e.evaluateCondition(expression)

	step.Status = "completed"
	step.Output = fmt.Sprintf("%t", result)
	db.DB.Save(step)

	e.variables[node.id] = result
	e.saveVariables()

	log.Printf("[Flow] Condition '%s': %s = %t", node.name, expression, result)
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
	for _, dep := range node.edgesIn {
		if s, ok := e.steps[dep]; ok {
			outputs = append(outputs, fmt.Sprintf("[%s]: %s", s.NodeName, s.Output))
		}
	}

	step.Status = "completed"
	step.Output = strings.Join(outputs, "\n---\n")
	db.DB.Save(step)

	e.variables[node.id] = step.Output
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
	for _, next := range node.edgesOut {
		if _, done := e.steps[next]; !done {
			step := &FlowStep{
				ExecID:    e.exec.ID,
				NodeID:    next,
				NodeName:  e.graph[next].name,
				NodeType:  e.graph[next].nodeType,
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
	for range node.edgesOut {
		// TODO: 根据 edge condition 判断是否跳过
		// 简化：跳过所有出边（实际应根据边的 condition 字段）
	}
}

// resolveTemplate 解析模板变量 ${node_id}
func (e *FlowExecutor) resolveTemplate(s string) string {
	for varName, varValue := range e.variables {
		placeholder := "${" + varName + "}"
		if strings.Contains(s, placeholder) {
			s = strings.ReplaceAll(s, placeholder, fmt.Sprintf("%v", varValue))
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
	language, _ := node.config["language"].(string)
	code, _ := node.config["code"].(string)
	timeoutMinutes, _ := node.config["timeout_minutes"].(float64)
	
	if code == "" {
		return fmt.Errorf("code node requires 'code' config")
	}
	
	log.Printf("[Flow] Executing code node: %s (language: %s)", node.name, language)
	
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
		e.variables[node.id] = step.Output
		e.saveVariables()
		return nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("code execution timed out after %v", timeout)
	}
}

// executeWebhookNode 执行 Webhook 节点 (HTTP 请求)
func (e *FlowExecutor) executeWebhookNode(node *dagNode, step *FlowStep) error {
	method, _ := node.config["method"].(string)
	url, _ := node.config["url"].(string)
	timeoutMinutes, _ := node.config["timeout_minutes"].(float64)
	
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
	
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	step.Output = fmt.Sprintf("Status: %d\nBody: %s", resp.StatusCode, string(body))
	
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned error status: %d", resp.StatusCode)
	}
	
	step.Status = "completed"
	e.variables[node.id] = step.Output
	e.saveVariables()
	return nil
}

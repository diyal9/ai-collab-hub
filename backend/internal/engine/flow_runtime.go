package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-collab-hub/internal/config"
	"ai-collab-hub/internal/db"
	"ai-collab-hub/internal/gateway"
	"ai-collab-hub/internal/model"
	"ai-collab-hub/internal/ws"
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
	var nodes []model.AgentNode
	if err := db.DB.Where("flow_id = ?", flow.ID).Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("load nodes: %w", err)
	}

	var edges []model.AgentEdge
	if err := db.DB.Where("flow_id = ?", flow.ID).Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("load edges: %w", err)
	}

	graph := make(map[string]*dagNode)
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

	for _, e := range edges {
		if src, ok := graph[e.SourceID]; ok {
			src.EdgesOut = append(src.EdgesOut, e.TargetID)
		}
		if tgt, ok := graph[e.TargetID]; ok {
			tgt.EdgesIn = append(tgt.EdgesIn, e.SourceID)
		}
	}

	if hasCycle(graph) {
		return nil, fmt.Errorf("flow contains cycle")
	}

	return graph, nil
}

func hasCycle(graph map[string]*dagNode) bool {
	visited := make(map[string]int)
	var dfs func(id string) bool
	dfs = func(id string) bool {
		if visited[id] == 1 { return true }
		if visited[id] == 2 { return false }
		visited[id] = 1
		if node, ok := graph[id]; ok {
			for _, next := range node.EdgesOut {
				if dfs(next) { return true }
			}
		}
		visited[id] = 2
		return false
	}
	for id := range graph {
		if dfs(id) { return true }
	}
	return false
}

func topologicalSort(graph map[string]*dagNode) []string {
	inDegree := make(map[string]int)
	for id, node := range graph {
		if _, ok := inDegree[id]; !ok { inDegree[id] = 0 }
		for _, out := range node.EdgesOut { inDegree[out]++ }
	}

	queue := []string{}
	for id, deg := range inDegree { if deg == 0 { queue = append(queue, id) } }
	sort.Strings(queue)

	result := []string{}
	totalNodes := len(graph)
	
	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]
		result = append(result, nodeID)

		if n, exists := graph[nodeID]; exists && n != nil {
			for _, next := range n.EdgesOut {
				if _, ok := inDegree[next]; ok {
					inDegree[next]--
					if inDegree[next] == 0 { queue = append(queue, next) }
				}
			}
		}
	}

	if len(result) < totalNodes { return nil }
	return result
}

// ─── 生产级引擎：FlowManager & Context ───

var Manager = &FlowManager{
    activeFlows: make(map[uint]*FlowExecutionCtx),
    mu:          sync.RWMutex{},
}

type FlowManager struct {
    activeFlows map[uint]*FlowExecutionCtx
    mu          sync.RWMutex
}

type FlowExecutionCtx struct {
    Exec      *FlowExecution
    Flow      model.AgentFlow
    Graph     map[string]*dagNode
    InDegree  map[string]int
    Variables map[string]interface{}
    Steps     map[string]*FlowStep
    
    ReadyQueue chan string
    Wg         sync.WaitGroup
    Mu         sync.Mutex
    Done       chan struct{}
    Paused     bool
    PauseCh    chan struct{}
    
    // Phase 1: Resilience
    Ctx         context.Context
    Cancel      context.CancelFunc
    RetryCounts map[string]int
}

func (fm *FlowManager) Get(execID uint) (*FlowExecutionCtx, bool) {
    fm.mu.RLock()
    defer fm.mu.RUnlock()
    ctx, ok := fm.activeFlows[execID]
    return ctx, ok
}

func (fm *FlowManager) Set(ctx *FlowExecutionCtx) {
    fm.mu.Lock()
    defer fm.mu.Unlock()
    fm.activeFlows[ctx.Exec.ID] = ctx
}

func (fm *FlowManager) Remove(execID uint) {
    fm.mu.Lock()
    defer fm.mu.Unlock()
    delete(fm.activeFlows, execID)
}

func StartFlow(flowID uint) (uint, error) {
	var flow model.AgentFlow
	if err := db.DB.First(&flow, flowID).Error; err != nil {
		return 0, fmt.Errorf("flow not found: %w", err)
	}

    graph, err := buildDAG(flow)
    if err != nil {
        return 0, fmt.Errorf("build DAG: %w", err)
    }

    order := topologicalSort(graph)
    if len(order) == 0 {
        return 0, fmt.Errorf("no nodes in flow")
    }

    exec := &FlowExecution{
        FlowID:    flow.ID,
        FlowName:  flow.Name,
        Status:    "running",
        Variables: "{}",
        StartedAt: time.Now(),
    }
    db.DB.Create(exec)

    inDegree := make(map[string]int)
    for id, node := range graph {
        if _, ok := inDegree[id]; !ok {
            inDegree[id] = 0
        }
        for _, out := range node.EdgesOut {
            inDegree[out]++
        }
    }

    ctx := &FlowExecutionCtx{
        Exec:        exec,
        Flow:        flow,
        Graph:       graph,
        InDegree:    inDegree,
        Variables:   make(map[string]interface{}),
        Steps:       make(map[string]*FlowStep),
        ReadyQueue:  make(chan string, 64),
        Done:        make(chan struct{}),
        PauseCh:     make(chan struct{}),
        RetryCounts: make(map[string]int),
    }
    ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
    
    Manager.Set(ctx)
    ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"flow_start","exec_id":%d,"flow_id":%d}`, exec.ID, flow.ID))

    // 初始化 ReadyQueue
    for _, nodeID := range order {
        if inDegree[nodeID] == 0 {
            ctx.ReadyQueue <- nodeID
        }
    }

    go ctx.Run()
    return exec.ID, nil
}

func RecoverFlows() {
    var runningExecs []FlowExecution
    db.DB.Where("status = ?", "running").Find(&runningExecs)

    for _, exec := range runningExecs {
        var flow model.AgentFlow
        if err := db.DB.First(&flow, exec.FlowID).Error; err != nil {
            log.Printf("[Flow] Recover failed: flow %d not found", exec.FlowID)
            continue
        }

        graph, err := buildDAG(flow)
        if err != nil {
            continue
        }

        inDegree := make(map[string]int)
        for id, node := range graph {
            if _, ok := inDegree[id]; !ok {
                inDegree[id] = 0
            }
            for _, out := range node.EdgesOut {
                inDegree[out]++
            }
        }

        variables := make(map[string]interface{})
        if exec.Variables != "" {
            json.Unmarshal([]byte(exec.Variables), &variables)
        }

        var steps []FlowStep
        db.DB.Where("exec_id = ?", exec.ID).Find(&steps)
        
        stepMap := make(map[string]*FlowStep)
        for i := range steps {
            stepMap[steps[i].NodeID] = &steps[i]
            // 已完成节点的下游入度应减 1
            node := graph[steps[i].NodeID]
            if node != nil && (steps[i].Status == "completed" || steps[i].Status == "skipped") {
                for _, out := range node.EdgesOut {
                    inDegree[out]--
                }
            }
        }

        ctx := &FlowExecutionCtx{
            Exec:        &exec,
            Flow:        flow,
            Graph:       graph,
            InDegree:    inDegree,
            Variables:   variables,
            Steps:       stepMap,
            ReadyQueue:  make(chan string, 64),
            Done:        make(chan struct{}),
            PauseCh:     make(chan struct{}),
            RetryCounts: make(map[string]int),
        }
        ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
        Manager.Set(ctx)

        // 重新计算 ReadyQueue
        for nodeID, deg := range inDegree {
            if deg == 0 && stepMap[nodeID] == nil {
                ctx.ReadyQueue <- nodeID
            }
        }
        
        go ctx.Run()
        log.Printf("[Flow] Recovered execution %d", exec.ID)
    }
}

func (ctx *FlowExecutionCtx) Run() {
    defer Manager.Remove(ctx.Exec.ID)
    
    for {
        select {
        case nodeID := <-ctx.ReadyQueue:
            ctx.Wg.Add(1)
            go ctx.executeNodeAsync(nodeID)
        case <-ctx.PauseCh:
            log.Printf("[Flow] Execution %d resumed", ctx.Exec.ID)
        case <-ctx.Ctx.Done():
            log.Printf("[Flow] Execution %d cancelled (Shutdown)", ctx.Exec.ID)
            return
        case <-ctx.Done:
            return
        }
    }
}

// Shutdown cancels all active flows gracefully
func Shutdown() {
    Manager.mu.RLock()
    defer Manager.mu.RUnlock()
    log.Printf("[Flow] Shutting down %d active flows...", len(Manager.activeFlows))
    for _, ctx := range Manager.activeFlows {
        ctx.Cancel()
    }
}

func (ctx *FlowExecutionCtx) executeNodeAsync(nodeID string) {
    defer ctx.Wg.Done()
    
    ctx.Mu.Lock()
    node := ctx.Graph[nodeID]
    ctx.Mu.Unlock()
    
    if node == nil || node.NodeType == "trigger" {
        ctx.onNodeComplete(nodeID)
        return
    }
    
    // Check upstream
    ctx.Mu.Lock()
    for _, dep := range node.EdgesIn {
        if step, ok := ctx.Steps[dep]; ok && (step.Status == "skipped" || step.Status == "failed") {
            ctx.createSkippedStep(nodeID, node)
            ctx.Mu.Unlock()
            ctx.onNodeComplete(nodeID)
            return
        }
    }
    ctx.Mu.Unlock()
    
    // Phase 1: Check Circuit Breaker
    if cb := GetCircuitBreaker(node.NodeType); cb != nil && !cb.Allow() {
        log.Printf("[Flow] Node %s blocked by circuit breaker", nodeID)
        // Retry later? For now, fail immediately to avoid pileup
        ctx.Mu.Lock()
        step := &FlowStep{
            ExecID: ctx.Exec.ID, NodeID: nodeID, NodeName: node.Name, NodeType: node.NodeType,
            Status: "failed", Error: "Circuit breaker open", Output: "Blocked by circuit breaker",
            StartedAt: time.Now(), EndedAt: time.Now(),
        }
        db.DB.Create(step)
        ctx.Steps[nodeID] = step
        ctx.Mu.Unlock()
        ctx.markDownstreamSkipped(nodeID)
        ctx.failExecution("Circuit breaker open")
        return
    }

    // Create running step
    step := &FlowStep{
        ExecID: ctx.Exec.ID, NodeID: nodeID, NodeName: node.Name, NodeType: node.NodeType,
        Status: "running", StartedAt: time.Now(),
    }
    db.DB.Create(step)
    ctx.Mu.Lock()
    ctx.Steps[nodeID] = step
    ctx.Mu.Unlock()
    
    ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"node_update","exec_id":%d,"node_id":"%s","status":"running"}`, ctx.Exec.ID, nodeID))

    // Phase 1: Timeout handling
    timeout := 10 * time.Minute // Default
    if node.Config != nil {
        if t, ok := node.Config["timeout_minutes"].(float64); ok && t > 0 {
            timeout = time.Duration(t) * time.Minute
        }
    }
    
    // Execute with timeout
    execCtx, execCancel := context.WithTimeout(ctx.Ctx, timeout)
    defer execCancel()
    
    errCh := make(chan error, 1)
    go func() {
        var err error
        switch node.NodeType {
        case "agent": err = ctx.executeAgentNode(node, step)
        case "condition": err = ctx.executeConditionNode(node, step)
        case "merge": err = ctx.executeMergeNode(node, step)
        case "code": err = ctx.executeCodeNode(node, step)
        case "sandbox": err = ctx.executeSandboxNode(node, step)
        case "webhook": err = ctx.executeWebhookNode(node, step)
        case "approval": err = ctx.executeApprovalNode(node, step)
        default: err = fmt.Errorf("unknown node type: %s", node.NodeType)
        }
        errCh <- err
    }()

    select {
    case err := <-errCh:
        // Execution finished (success or error)
        if err != nil {
            if IsApprovalPending(err) {
                ctx.Exec.Status = "waiting_approval"
                ctx.Exec.PendingNodeID = nodeID
                ctx.Exec.PendingAt = time.Now()
                db.DB.Save(ctx.Exec)
                ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"flow_pause","exec_id":%d,"node_id":"%s"}`, ctx.Exec.ID, nodeID))
                return
            }
            ctx.handleNodeFailure(nodeID, node, step, err)
        } else {
            // Phase 1: Record Success
            if cb := GetCircuitBreaker(node.NodeType); cb != nil {
                cb.RecordSuccess()
            }
            ctx.onNodeComplete(nodeID)
        }
        
    case <-execCtx.Done():
        // Phase 1: Timeout
        errMsg := "Execution timed out"
        if ctx.Ctx.Err() != nil {
            errMsg = "Flow cancelled"
        }
        log.Printf("[Flow] Node %s %s", nodeID, errMsg)
        
        ctx.Mu.Lock()
        step.Status = "failed"
        step.Error = errMsg
        step.EndedAt = time.Now()
        ctx.Mu.Unlock()
        db.DB.Save(step)
        
        if cb := GetCircuitBreaker(node.NodeType); cb != nil {
            cb.RecordFailure() // Timeout counts as failure
        }
        // Retry on timeout? Yes, if retryable
        ctx.handleNodeFailure(nodeID, node, step, fmt.Errorf(errMsg))
    }
}

// handleNodeFailure implements Phase 1: Retry Logic
func (ctx *FlowExecutionCtx) handleNodeFailure(nodeID string, node *dagNode, step *FlowStep, err error) {
    ctx.Mu.Lock()
    retryCount := ctx.RetryCounts[nodeID]
    maxRetries := 3
    if node.Config != nil {
        if r, ok := node.Config["retry_count"].(float64); ok {
            maxRetries = int(r)
        }
    }
    
    isRetryable := true
    // 4xx errors are not retryable (check error string or type)
    if strings.Contains(err.Error(), "status: 4") {
        isRetryable = false
    }
    
    ctx.Mu.Unlock()

    if isRetryable && retryCount < maxRetries {
        ctx.RetryCounts[nodeID] = retryCount + 1
        delay := time.Duration(retryCount+1) * time.Second // Simple backoff
        log.Printf("[Flow] Retrying node %s (attempt %d/%d) in %v", nodeID, retryCount+1, maxRetries, delay)
        
        // Update step status to retrying
        ctx.Mu.Lock()
        step.Status = "retrying"
        step.Output = fmt.Sprintf("Retrying... (%d/%d)", retryCount+1, maxRetries)
        ctx.Mu.Unlock()
        db.DB.Save(step)
        
        time.AfterFunc(delay, func() {
            ctx.ReadyQueue <- nodeID
        })
    } else {
        // Final failure
        log.Printf("[Flow] Node %s failed permanently: %v", nodeID, err)
        ctx.Mu.Lock()
        step.Status = "failed"
        step.Error = err.Error()
        step.EndedAt = time.Now()
        ctx.Mu.Unlock()
        db.DB.Save(step)
        
        ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"node_update","exec_id":%d,"node_id":"%s","status":"failed","error":"%s"}`, ctx.Exec.ID, nodeID, err.Error()))
        
        if cb := GetCircuitBreaker(node.NodeType); cb != nil {
            cb.RecordFailure()
        }
        ctx.markDownstreamSkipped(nodeID)
        ctx.failExecution(err.Error())
    }
}

func (ctx *FlowExecutionCtx) onNodeComplete(nodeID string) {
    ctx.Mu.Lock()
    defer ctx.Mu.Unlock()
    
    step := ctx.Steps[nodeID]
    if step == nil { return }
    if step.Status == "running" {
        step.Status = "completed"
        step.EndedAt = time.Now()
        db.DB.Save(step)
        
        ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"node_update","exec_id":%d,"node_id":"%s","status":"completed"}`, ctx.Exec.ID, nodeID))
    }
    
    // 更新下游入度
    node := ctx.Graph[nodeID]
    if node != nil {
        for _, out := range node.EdgesOut {
            ctx.InDegree[out]--
            if ctx.InDegree[out] == 0 {
                if _, done := ctx.Steps[out]; !done {
                    ctx.ReadyQueue <- out
                }
            }
        }
    }
    
    // 检查是否全部完成
    allDone := true
    hasPending := false
    for _, step := range ctx.Steps {
        if step.Status == "running" || step.Status == "pending" {
            hasPending = true
        }
    }
    
    if !hasPending {
        for _, deg := range ctx.InDegree {
            if deg > 0 { allDone = false; break }
        }
    }
    
    if allDone && !hasPending {
        ctx.Exec.Status = "completed"
        ctx.Exec.EndedAt = time.Now()
        db.DB.Save(ctx.Exec)
        ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"flow_complete","exec_id":%d}`, ctx.Exec.ID))
        close(ctx.Done)
    }
}

func (ctx *FlowExecutionCtx) createSkippedStep(nodeID string, node *dagNode) {
    step := &FlowStep{
        ExecID:   ctx.Exec.ID,
        NodeID:   nodeID,
        NodeName: node.Name,
        NodeType: node.NodeType,
        Status:   "skipped",
        Output:   "Skipped due to upstream failure",
    }
    db.DB.Create(step)
    ctx.Steps[nodeID] = step
    ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"node_update","exec_id":%d,"node_id":"%s","status":"skipped"}`, ctx.Exec.ID, nodeID))
}

func (ctx *FlowExecutionCtx) markDownstreamSkipped(nodeID string) {
    ctx.Mu.Lock()
    defer ctx.Mu.Unlock()
    
    node := ctx.Graph[nodeID]
    if node == nil { return }
    
    for _, next := range node.EdgesOut {
        if _, done := ctx.Steps[next]; !done {
            step := &FlowStep{
                ExecID:   ctx.Exec.ID,
                NodeID:   next,
                NodeName: ctx.Graph[next].Name,
                NodeType: ctx.Graph[next].NodeType,
                Status:   "skipped",
                Output:   "Skipped due to upstream failure",
            }
            db.DB.Create(step)
            ctx.Steps[next] = step
            ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"node_update","exec_id":%d,"node_id":"%s","status":"skipped"}`, ctx.Exec.ID, next))
            ctx.markDownstreamSkipped(next)
        }
    }
}

func (ctx *FlowExecutionCtx) failExecution(errMsg string) {
    ctx.Mu.Lock()
    defer ctx.Mu.Unlock()
    
    ctx.Exec.Status = "failed"
    ctx.Exec.Error = errMsg
    ctx.Exec.EndedAt = time.Now()
    db.DB.Save(ctx.Exec)
    ws.Hub.Broadcast <- []byte(fmt.Sprintf(`{"type":"flow_fail","exec_id":%d,"error":"%s"}`, ctx.Exec.ID, errMsg))
    close(ctx.Done)
}

func ResumeFlowExecution(execID uint) error {
    ctx, ok := Manager.Get(execID)
    if !ok {
        return fmt.Errorf("execution not found or not active")
    }
    
    ctx.Mu.Lock()
    if ctx.Exec.Status != "waiting_approval" {
        ctx.Mu.Unlock()
        return fmt.Errorf("execution is not waiting for approval")
    }
    
    pendingNodeID := ctx.Exec.PendingNodeID
    ctx.Exec.Status = "running"
    ctx.Exec.PendingNodeID = ""
    ctx.Exec.PendingAt = time.Time{}
    ctx.Mu.Unlock()
    
    // 标记审批节点为完成
    ctx.Mu.Lock()
    if step, ok := ctx.Steps[pendingNodeID]; ok {
        step.Status = "completed"
        step.Output = "Approved by user."
        step.EndedAt = time.Now()
        ctx.Mu.Unlock()
        db.DB.Save(step)
        
        ctx.Variables[pendingNodeID] = "approved"
        ctx.saveVariables()
        
        // 继续流程
        ctx.onNodeComplete(pendingNodeID)
    } else {
        ctx.Mu.Unlock()
    }
    
    select {
    case ctx.PauseCh <- struct{}{}:
    default:
    }
    
    return nil
}

func AbortFlowExecution(execID uint) error {
    ctx, ok := Manager.Get(execID)
    if !ok {
        return fmt.Errorf("execution not found or not active")
    }
    
    ctx.Mu.Lock()
    if ctx.Exec.Status != "waiting_approval" {
        ctx.Mu.Unlock()
        return fmt.Errorf("execution is not waiting for approval")
    }
    
    pendingNodeID := ctx.Exec.PendingNodeID
    
    ctx.Exec.Status = "failed"
    ctx.Exec.Error = "Approval rejected by user"
    ctx.Exec.PendingNodeID = ""
    ctx.Exec.EndedAt = time.Now()
    ctx.Mu.Unlock()
    
    // 标记审批节点和运行中节点为失败/跳过
    if pendingNodeID != "" {
        ctx.Mu.Lock()
        if step, ok := ctx.Steps[pendingNodeID]; ok {
            step.Status = "failed"
            step.Output = "Rejected by user."
            ctx.Mu.Unlock()
            db.DB.Save(step)
        } else {
            ctx.Mu.Unlock()
        }
    }
    
    ctx.markDownstreamSkipped(pendingNodeID)
    
    close(ctx.Done)
    return nil
}

func (ctx *FlowExecutionCtx) executeAgentNode(node *dagNode, step *FlowStep) error {
	agentIDStr, _ := node.Config["agent_id"].(string)
	prompt, _ := node.Config["prompt"].(string)
	requiredSkills, _ := node.Config["required_skills"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)

	// 模板变量替换
	prompt = ctx.resolveTemplate(prompt)

	// 创建 Agent 任务
	var agentID uint
	fmt.Sscanf(agentIDStr, "%d", &agentID)

	taskID := fmt.Sprintf("flow_%d_node_%s", ctx.Exec.ID, node.ID)
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
			ctx.Variables[node.ID] = lastLog.Content
			ctx.Variables[node.ID+"_task_id"] = taskID
			ctx.saveVariables()
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

func (ctx *FlowExecutionCtx) executeConditionNode(node *dagNode, step *FlowStep) error {
	expression, _ := node.Config["expression"].(string)

	// 解析表达式（简单实现：支持 variable == value 格式）
	expression = ctx.resolveTemplate(expression)

	result := ctx.evaluateCondition(expression)

	step.Status = "completed"
	step.Output = fmt.Sprintf("%t", result)
	db.DB.Save(step)

	ctx.Variables[node.ID] = result
	ctx.saveVariables()

	log.Printf("[Flow] Condition '%s': %s = %t", node.Name, expression, result)
	return nil
}

func (ctx *FlowExecutionCtx) evaluateCondition(expr string) bool {
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

func (ctx *FlowExecutionCtx) executeMergeNode(node *dagNode, step *FlowStep) error {
	// 汇聚节点：等待所有上游完成
	var outputs []string
	for _, dep := range node.EdgesIn {
		if s, ok := ctx.Steps[dep]; ok {
			outputs = append(outputs, fmt.Sprintf("[%s]: %s", s.NodeName, s.Output))
		}
	}

	step.Status = "completed"
	step.Output = strings.Join(outputs, "\n---\n")
	db.DB.Save(step)

	ctx.Variables[node.ID] = step.Output
	ctx.saveVariables()

	return nil
}

func (ctx *FlowExecutionCtx) executeTriggerNode(node *dagNode, step *FlowStep) error {
	// Trigger 节点：流程起点，直接完成
	step.Status = "completed"
	step.Output = "Trigger activated"
	db.DB.Save(step)
	return nil
}

// skipFalseBranch 条件为 false 时，跳过不匹配的分支
func (ctx *FlowExecutionCtx) skipFalseBranch(condNodeID string) {
	node := ctx.Graph[condNodeID]
	for range node.EdgesOut {
		// TODO: 根据 edge condition 判断是否跳过
		// 简化：跳过所有出边（实际应根据边的 condition 字段）
	}
}

// resolveTemplate 解析模板变量 ${node_id}
func (ctx *FlowExecutionCtx) resolveTemplate(s string) string {
	for varName, varValue := range ctx.Variables {
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

func (ctx *FlowExecutionCtx) saveVariables() {
	data, _ := json.Marshal(ctx.Variables)
	ctx.Exec.Variables = string(data)
	db.DB.Model(ctx.Exec).Update("variables", string(data))
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
func (ctx *FlowExecutionCtx) executeCodeNode(node *dagNode, step *FlowStep) error {
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
		ctx.Variables[node.ID] = step.Output
		ctx.saveVariables()
		return nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("code execution timed out after %v", timeout)
	}
}

// executeSandboxNode 通过 Bridge 调度 Sandbox 执行任务 (同步阻塞)
func (ctx *FlowExecutionCtx) executeSandboxNode(node *dagNode, step *FlowStep) error {
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
		for k, v := range ctx.Variables {
			variables[k] = fmt.Sprintf("%v", v)
		}
		resolved, err := ResolvePromptTemplate(tmplName, variables)
		if err != nil {
			return fmt.Errorf("resolve prompt template '%s': %w", tmplName, err)
		}
		command = resolved
	}

	// 模板变量替换
	command = ctx.resolveTemplate(command)

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
		"task_id":    fmt.Sprintf("flow_%d_node_%s", ctx.Exec.ID, node.ID),
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
	ctx.Variables[node.ID] = result.Output
	ctx.saveVariables()
	return nil
}

// executeWebhookNode 执行 Webhook 节点 (HTTP 请求)
func (ctx *FlowExecutionCtx) executeWebhookNode(node *dagNode, step *FlowStep) error {
	method, _ := node.Config["method"].(string)
	url, _ := node.Config["url"].(string)
	timeoutMinutes, _ := node.Config["timeout_minutes"].(float64)
	headersRaw, _ := node.Config["headers"].(string)
	bodyRaw, _ := node.Config["body"].(string)
	
	if url == "" {
		return fmt.Errorf("webhook node requires 'url' config")
	}
	
	url = ctx.resolveTemplate(url)
	
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
		bodyRaw = ctx.resolveTemplate(bodyRaw)
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
	ctx.Variables[node.ID] = step.Output
	ctx.saveVariables()
	return nil
}

// executeApprovalNode 人在回路：发送飞书审批卡片并暂停流程
func (ctx *FlowExecutionCtx) executeApprovalNode(node *dagNode, step *FlowStep) error {
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
	cardBody = ctx.resolveTemplate(cardBody)
	cardTitle = ctx.resolveTemplate(cardTitle)

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
						"url":  fmt.Sprintf("%s/api/flows/%d/executions/%d/approve?token=%s", getPublicURL(), ctx.Exec.FlowID, ctx.Exec.ID, generateApprovalToken(ctx.Exec.ID)),
					},
					map[string]interface{}{
						"tag":  "button",
						"text": map[string]interface{}{"tag": "plain_text", "content": "❌ 拒绝"},
						"type": "danger",
						"url":  fmt.Sprintf("%s/api/flows/%d/executions/%d/reject?token=%s", getPublicURL(), ctx.Exec.FlowID, ctx.Exec.ID, generateApprovalToken(ctx.Exec.ID)),
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

	ctx.Exec.Status = "waiting_approval"
	ctx.Exec.PendingNodeID = node.ID
	ctx.Exec.PendingAt = time.Now()
	ctx.Exec.ApprovalTimeout = int(timeoutMinutes)
	if ctx.Exec.ApprovalTimeout == 0 {
		ctx.Exec.ApprovalTimeout = 1440 // 默认 24 小时
	}
	db.DB.Save(ctx.Exec)

	// 返回特殊 error 让执行器暂停（不标记为 failed）
	return &ApprovalPendingError{ExecID: ctx.Exec.ID}
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

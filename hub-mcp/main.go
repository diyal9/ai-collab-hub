package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─── Configuration ───

var (
	hubURL  string
	hubToken string
	client  *http.Client
)

func init() {
	hubURL = os.Getenv("HUB_URL")
	if hubURL == "" {
		hubURL = "http://localhost:8087"
	}
	hubToken = os.Getenv("HUB_TOKEN")
	client = &http.Client{Timeout: 30 * time.Second}
}

// ─── JSON-RPC 2.0 Types ───

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ─── Tool Definitions ───

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema struct {
		Type       string              `json:"type"`
		Properties map[string]Property `json:"properties,omitempty"`
		Required   []string            `json:"required,omitempty"`
	} `json:"inputSchema"`
}

type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

var tools = []Tool{
	{
		Name:        "list_tasks",
		Description: "List tasks from AI Collab Hub. Filter by status and limit results.",
		InputSchema: buildSchema(map[string]Property{
			"status": {"string", "Filter by status (queued, running, completed, failed)"},
			"limit":  {"number", "Maximum number of tasks to return (default: 20)"},
		}, nil),
	},
	{
		Name:        "get_task",
		Description: "Get details of a specific task including logs.",
		InputSchema: buildSchema(map[string]Property{
			"task_id": {"string", "The task ID to look up"},
		}, []string{"task_id"}),
	},
	{
		Name:        "create_task",
		Description: "Create a new task in AI Collab Hub for an agent to execute.",
		InputSchema: buildSchema(map[string]Property{
			"title":           {"string", "Task title"},
			"prompt":          {"string", "Task instruction/prompt for the agent"},
			"agent_id":        {"number", "Optional specific agent ID to assign"},
			"required_skills": {"string", "Optional JSON array of required skills, e.g. [\"terminal\", \"file\"]"},
			"parent_task_id":  {"string", "Optional parent task ID for context chain"},
			"timeout_minutes": {"number", "Task timeout in minutes (default: 30)"},
		}, []string{"title", "prompt"}),
	},
	{
		Name:        "complete_task",
		Description: "Mark a task as completed with output. Use this when you've finished working on an assigned task.",
		InputSchema: buildSchema(map[string]Property{
			"task_id": {"string", "The task ID to complete"},
			"output":  {"string", "The output/result of the task"},
		}, []string{"task_id", "output"}),
	},
	{
		Name:        "list_flows",
		Description: "List all workflow flows in AI Collab Hub.",
		InputSchema: buildSchema(nil, nil),
	},
	{
		Name:        "execute_flow",
		Description: "Execute a workflow flow by ID. This starts the DAG execution engine.",
		InputSchema: buildSchema(map[string]Property{
			"flow_id": {"number", "The flow ID to execute"},
		}, []string{"flow_id"}),
	},
	{
		Name:        "get_agent_instances",
		Description: "List all registered agent instances with their status and capabilities.",
		InputSchema: buildSchema(nil, nil),
	},
}

func buildSchema(props map[string]Property, required []string) struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
} {
	s := struct {
		Type       string              `json:"type"`
		Properties map[string]Property `json:"properties,omitempty"`
		Required   []string            `json:"required,omitempty"`
	}{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
	return s
}

// ─── Hub API Client ───

func apiRequest(method, path string, payload interface{}) (interface{}, error) {
	url := fmt.Sprintf("%s%s", hubURL, path)
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = strings.NewReader(string(data))
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if hubToken != "" {
		req.Header.Set("Authorization", "Bearer "+hubToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// ─── Tool Handlers ───

func handleListTasks(args map[string]interface{}) (string, error) {
	result, err := apiRequest("GET", "/api/agent-tasks", nil)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch tasks: %v", err)
	}

	tasks, ok := result.([]interface{})
	if !ok {
		return "", fmt.Errorf("Unexpected response format")
	}

	// Apply filters
	if status, ok := args["status"].(string); ok && status != "" {
		filtered := []interface{}{}
		for _, t := range tasks {
			if task, ok := t.(map[string]interface{}); ok {
				if task["status"] == status {
					filtered = append(filtered, t)
				}
			}
		}
		tasks = filtered
	}

	// Apply limit
	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if len(tasks) > limit {
		tasks = tasks[:limit]
	}

	// Simplify output
	output := []map[string]interface{}{}
	for _, t := range tasks {
		if task, ok := t.(map[string]interface{}); ok {
			output = append(output, map[string]interface{}{
				"id":             task["id"],
				"title":          task["title"],
				"status":         task["status"],
				"agent_id":       task["agent_id"],
				"parent_task_id": task["parent_task_id"],
				"created_at":     task["created_at"],
			})
		}
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"tasks": output,
		"count": len(output),
	}, "", "  ")
	return string(data), nil
}

func handleGetTask(args map[string]interface{}) (string, error) {
	taskID, ok := args["task_id"].(string)
	if !ok {
		return "", fmt.Errorf("task_id is required")
	}

	task, err := apiRequest("GET", fmt.Sprintf("/api/agent-tasks/%s", taskID), nil)
	if err != nil {
		return "", fmt.Errorf("Task not found: %v", err)
	}

	logs, _ := apiRequest("GET", fmt.Sprintf("/api/agent-tasks/%s/logs", taskID), nil)
	if logsArr, ok := logs.([]interface{}); ok && len(logsArr) > 50 {
		logs = logsArr[:50]
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"task": task,
		"logs": logs,
	}, "", "  ")
	return string(data), nil
}

func handleCreateTask(args map[string]interface{}) (string, error) {
	title, _ := args["title"].(string)
	prompt, _ := args["prompt"].(string)
	if title == "" || prompt == "" {
		return "", fmt.Errorf("title and prompt are required")
	}

	payload := map[string]interface{}{
		"title":    title,
		"prompt":   prompt,
		"priority": 1,
	}

	if agentID, ok := args["agent_id"].(float64); ok {
		payload["agent_id"] = int(agentID)
	}
	if skills, ok := args["required_skills"].(string); ok {
		payload["required_skills"] = skills
	}
	if parentID, ok := args["parent_task_id"].(string); ok {
		payload["parent_task_id"] = parentID
	}
	if timeout, ok := args["timeout_minutes"].(float64); ok {
		payload["timeout_minutes"] = int(timeout)
	} else {
		payload["timeout_minutes"] = 30
	}

	result, err := apiRequest("POST", "/api/agent-tasks", payload)
	if err != nil {
		return "", fmt.Errorf("Failed to create task: %v", err)
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"message": "Task created successfully",
		"task":    result,
	}, "", "  ")
	return string(data), nil
}

func handleCompleteTask(args map[string]interface{}) (string, error) {
	taskID, _ := args["task_id"].(string)
	output, _ := args["output"].(string)
	if taskID == "" || output == "" {
		return "", fmt.Errorf("task_id and output are required")
	}

	payload := map[string]interface{}{
		"task_id": taskID,
		"status":  "completed",
		"output":  output,
	}

	result, err := apiRequest("POST", "/api/agent/callback", payload)
	if err != nil {
		return "", fmt.Errorf("Failed to complete task: %v", err)
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"message": fmt.Sprintf("Task %s marked as completed", taskID),
		"result":  result,
	}, "", "  ")
	return string(data), nil
}

func handleListFlows(args map[string]interface{}) (string, error) {
	result, err := apiRequest("GET", "/api/flows", nil)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch flows: %v", err)
	}

	flows, ok := result.([]interface{})
	if !ok {
		return "", fmt.Errorf("Unexpected response format")
	}

	output := []map[string]interface{}{}
	for _, f := range flows {
		if flow, ok := f.(map[string]interface{}); ok {
			output = append(output, map[string]interface{}{
				"id":          flow["id"],
				"name":        flow["name"],
				"description": flow["description"],
				"type":        flow["type"],
				"status":      flow["status"],
			})
		}
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"flows": output,
		"count": len(output),
	}, "", "  ")
	return string(data), nil
}

func handleExecuteFlow(args map[string]interface{}) (string, error) {
	flowID, ok := args["flow_id"].(float64)
	if !ok {
		return "", fmt.Errorf("flow_id is required")
	}

	result, err := apiRequest("POST", fmt.Sprintf("/api/flows/%d/execute", int(flowID)), nil)
	if err != nil {
		return "", fmt.Errorf("Failed to execute flow: %v", err)
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"message": fmt.Sprintf("Flow %d execution started", int(flowID)),
		"result":  result,
	}, "", "  ")
	return string(data), nil
}

func handleGetAgentInstances(args map[string]interface{}) (string, error) {
	result, err := apiRequest("GET", "/api/agent-instances", nil)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch agents: %v", err)
	}

	agents, ok := result.([]interface{})
	if !ok {
		return "", fmt.Errorf("Unexpected response format")
	}

	output := []map[string]interface{}{}
	for _, a := range agents {
		if agent, ok := a.(map[string]interface{}); ok {
			output = append(output, map[string]interface{}{
				"id":           agent["id"],
				"name":         agent["name"],
				"type":         agent["type"],
				"status":       agent["status"],
				"capabilities": agent["capabilities"],
				"platform":     agent["platform"],
			})
		}
	}

	data, _ := json.MarshalIndent(map[string]interface{}{
		"agents": output,
		"count":  len(output),
	}, "", "  ")
	return string(data), nil
}

// ─── Request Router ───

var toolHandlers = map[string]func(map[string]interface{}) (string, error){
	"list_tasks":          handleListTasks,
	"get_task":            handleGetTask,
	"create_task":         handleCreateTask,
	"complete_task":       handleCompleteTask,
	"list_flows":          handleListFlows,
	"execute_flow":        handleExecuteFlow,
	"get_agent_instances": handleGetAgentInstances,
}

func handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "ai-collab-hub",
					"version": "1.0.0",
				},
			},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": tools,
			},
		}

	case "tools/call":
		var params CallToolParams
		if req.Params != nil {
			json.Unmarshal(req.Params, &params)
		}

		handler, exists := toolHandlers[params.Name]
		if !exists {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32601,
					Message: fmt.Sprintf("Unknown tool: %s", params.Name),
				},
			}
		}

		result, err := handler(params.Arguments)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32603,
					Message: err.Error(),
				},
			}
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": result,
					},
				},
			},
		}

	case "ping":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// ─── Main Loop ───

func main() {
	log.SetOutput(os.Stderr)
	log.Println("MCP Server starting...")
	log.Printf("Hub URL: %s", hubURL)
	if hubToken != "" {
		log.Println("Token: configured")
	} else {
		log.Println("Token: not configured (anonymous access)")
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			resp := &JSONRPCResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			}
			sendResponse(resp)
			continue
		}

		resp := handleRequest(req)
		sendResponse(resp)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
}

func sendResponse(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}
	fmt.Println(string(data))
}

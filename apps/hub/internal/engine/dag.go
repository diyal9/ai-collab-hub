package engine

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
)

// DAG represents a workflow definition
type DAG struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

// Node represents a single step in the workflow
type Node struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"` // agent, condition, merge, trigger
	Config map[string]interface{} `json:"config"`
	Status string                 `json:"status"` // pending, running, completed, failed
	Result string                 `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

// Edge represents a connection between nodes
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"` // For condition branches
}

// Runner executes the DAG
type Runner struct {
	DAG         *DAG
	Context     map[string]string // Shared context (variable passing)
	Mu          sync.Mutex
	NodeHandler func(node *Node, ctx map[string]string) (string, error)
}

// NewRunner creates a new DAG runner
func NewRunner(dagJSON []byte, handler func(node *Node, ctx map[string]string) (string, error)) (*Runner, error) {
	var dag DAG
	if err := json.Unmarshal(dagJSON, &dag); err != nil {
		return nil, fmt.Errorf("invalid DAG JSON: %w", err)
	}

	return &Runner{
		DAG:         &dag,
		Context:     make(map[string]string),
		NodeHandler: handler,
	}, nil
}

// Execute runs the DAG in topological order
func (r *Runner) Execute() error {
	// 1. Topological Sort
	order, err := r.topologicalSort()
	if err != nil {
		return fmt.Errorf("cycle detected or invalid DAG: %w", err)
	}

	// 2. Execute nodes in order
	// Note: In a real implementation, this should handle parallel execution for independent nodes.
	// For now, we do sequential execution to ensure context passing works.
	for _, nodeID := range order {
		node := r.getNode(nodeID)
		if node == nil {
			continue
		}

		log.Printf("[DAG] Executing node: %s (%s)", node.ID, node.Type)
		node.Status = "running"

		// Execute the node handler (e.g., call Aider, run script, etc.)
		result, err := r.NodeHandler(node, r.Context)
		
		r.Mu.Lock()
		if err != nil {
			node.Status = "failed"
			node.Error = err.Error()
			r.Mu.Unlock()
			return fmt.Errorf("node %s failed: %w", node.ID, err)
		}

		node.Status = "completed"
		node.Result = result
		// Store result in context for downstream nodes
		r.Context[node.ID+"_output"] = result
		r.Mu.Unlock()
	}

	return nil
}

func (r *Runner) getNode(id string) *Node {
	for _, n := range r.DAG.Nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

func (r *Runner) topologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	graph := make(map[string][]string)
	nodes := make(map[string]bool)

	for _, n := range r.DAG.Nodes {
		nodes[n.ID] = true
		inDegree[n.ID] = 0
	}

	for _, e := range r.DAG.Edges {
		graph[e.Source] = append(graph[e.Source], e.Target)
		inDegree[e.Target]++
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue) // Deterministic order

	var result []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		result = append(result, curr)

		for _, neighbor := range graph[curr] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
				sort.Strings(queue)
			}
		}
	}

	if len(result) != len(nodes) {
		return nil, fmt.Errorf("cycle detected")
	}

	return result, nil
}

// 本地开发用 Mock Hub API（无生产 Hub 源码时的最小替身）
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
)

const apiPrefix = "/ai-collab-hub/api"

func main() {
	port := os.Getenv("MOCK_HUB_PORT")
	if port == "" {
		port = "8085"
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, map[string]string{"status": "ok", "service": "mock-hub"})
	})
	mux.HandleFunc("/resources", handleResources)
	mux.HandleFunc("/resources/", handleResources)
	mux.HandleFunc(apiPrefix+"/", handleAPI)
	mux.HandleFunc(apiPrefix, handleAPI)

	addr := ":" + port
	log.Printf("mock-hub listening on %s (prefix %s)", addr, apiPrefix)
	log.Fatal(http.ListenAndServe(addr, cors(mux)))
}

func handleAPI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, apiPrefix)
	if path == "" {
		path = "/"
	}

	switch {
	case path == "/login" && r.Method == http.MethodPost:
		jsonOK(w, map[string]string{"token": "mock-jwt-dev", "role": "admin", "user_id": "1"})
		return
	case path == "/agents":
		jsonOK(w, map[string]interface{}{
			"online_count": 2,
			"agents": []map[string]string{
				{"name": "local-bridge", "status": "online"},
				{"name": "cursor-main", "status": "online"},
			},
		})
		return
	case path == "/flows" || strings.HasPrefix(path, "/flows/"):
		handleFlows(w, r, path)
		return
	case path == "/knowledge" || strings.HasPrefix(path, "/knowledge/"):
		handleKnowledge(w, r, path)
		return
	case path == "/tasks" || strings.HasPrefix(path, "/tasks/"):
		handleTasks(w, r, path)
		return
	case path == "/files" || strings.HasPrefix(path, "/files/"):
		handleFiles(w, r, path)
		return
	case path == "/upload" && r.Method == http.MethodPost:
		jsonOK(w, map[string]interface{}{"id": 1, "file_name": "uploaded.bin", "size": 0})
		return
	case path == "/terminals" || strings.HasPrefix(path, "/terminals/"):
		jsonOK(w, emptyList())
		return
	case path == "/users":
		jsonOK(w, []map[string]string{{"id": "1", "username": "admin", "role": "admin"}})
		return
	case path == "/memory-sessions":
		jsonOK(w, emptyList())
		return
	case path == "/agent-instances" || strings.HasPrefix(path, "/agent-instances/"):
		jsonOK(w, []map[string]interface{}{
			{
				"id": 1, "name": "cursor-main", "type": "cursor",
				"platform": "windows", "host": "dev-machine", "status": "online",
				"capabilities": `["code","review"]`,
			},
		})
		return
	case path == "/prompt-templates" || strings.HasPrefix(path, "/prompt-templates/"):
		jsonOK(w, emptyList())
		return
	case strings.HasPrefix(path, "/flow-executions/"):
		jsonOK(w, emptyList())
		return
	}

	writeFallback(w, r)
}

func handleFlows(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}
	if path == "/flows" {
		jsonOK(w, []map[string]string{
			{"id": "1", "name": "Demo Flow", "status": "active"},
			{"id": "2", "name": "Draft Flow", "status": "draft"},
		})
		return
	}
	jsonOK(w, emptyList())
}

func handleKnowledge(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		jsonOK(w, map[string]bool{"ok": true})
		return
	}
	if path == "/knowledge/stats" {
		jsonOK(w, map[string]int{"pending": 1, "approved": 0, "rejected": 0})
		return
	}
	if path == "/knowledge" {
		status := r.URL.Query().Get("status")
		items := []map[string]string{
			{"id": "1", "title": "待审条目", "status": "pending"},
			{"id": "2", "title": "已发布指南", "status": "approved"},
		}
		if status != "" {
			filtered := make([]map[string]string, 0)
			for _, item := range items {
				if item["status"] == status {
					filtered = append(filtered, item)
				}
			}
			jsonOK(w, filtered)
			return
		}
		jsonOK(w, items)
		return
	}
	jsonOK(w, emptyList())
}

func handleTasks(w http.ResponseWriter, r *http.Request, path string) {
	if path != "/tasks" {
		jsonOK(w, emptyList())
		return
	}
	if r.Method == http.MethodGet {
		jsonOK(w, []map[string]string{
			{"id": "t1", "title": "Agent Team 流水线验收", "type": "qa", "status": "running"},
			{"id": "t2", "title": "Dashboard 团队入口", "type": "frontend", "status": "completed"},
		})
		return
	}
	jsonOK(w, map[string]string{"id": "task_mock", "status": "pending"})
}

func handleFiles(w http.ResponseWriter, r *http.Request, path string) {
	switch {
	case path == "/files" && r.Method == http.MethodGet:
		jsonOK(w, map[string]interface{}{
			"database_files": emptyList(),
			"legacy_files":   emptyList(),
			"uploaded_files": emptyList(),
		})
	case path == "/files/download":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mock file content"))
	case strings.HasPrefix(path, "/files/") && r.Method == http.MethodDelete:
		jsonOK(w, map[string]bool{"ok": true})
	default:
		writeFallback(w, r)
	}
}

func handleResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeFallback(w, r)
		return
	}
	jsonOK(w, emptyList())
}

func writeFallback(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		jsonOK(w, emptyList())
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		jsonOK(w, map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func emptyList() []interface{} {
	return []interface{}{}
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/ai-collab-hub/") || r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/resources") {
			next.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

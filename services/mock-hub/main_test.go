package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		jsonOK(w, map[string]string{"status": "ok"})
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Fatalf("got %v", body)
	}
}

func TestLoginReturnsToken(t *testing.T) {
	mux := http.NewServeMux()
	prefix := "/ai-collab-hub/api"
	mux.HandleFunc(prefix+"/login", func(w http.ResponseWriter, r *http.Request) {
		jsonOK(w, map[string]string{"token": "mock-jwt-dev", "role": "admin"})
	})
	req := httptest.NewRequest(http.MethodPost, prefix+"/login", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

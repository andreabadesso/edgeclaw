// Package registry provides a local tool registry and HTTP API that exposes
// EdgeClaw's IoT tools to PicoClaw. PicoClaw calls tools as external HTTP
// endpoints; this package bridges the gap by holding tool instances in memory
// and serving them over a JSON API.
//
// Endpoints:
//
//	GET  /tools                   - list all registered tools (name + description)
//	POST /tools/{name}/execute   - invoke a tool with JSON arguments
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Tool is the interface that every EdgeClaw tool implements. It mirrors
// PicoClaw's tool contract so tools can be used locally or forwarded
// over HTTP without adaptation.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]string) (string, error)
}

// Registry holds a set of named tools and provides lookup by name.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// New creates an empty tool registry.
func New() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry. If a tool with the same name already
// exists it is silently replaced.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
	log.Printf("[registry] registered tool: %s", t.Name())
}

// Get returns a tool by name, or nil if not found.
func (r *Registry) Get(name string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools[name]
}

// All returns a snapshot of every registered tool.
func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// ---------- HTTP API types ----------

// toolInfo is the JSON representation of a tool in the /tools listing.
type toolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// executeRequest is the JSON body expected by /tools/{name}/execute.
type executeRequest struct {
	Args map[string]string `json:"args"`
}

// executeResponse is the JSON body returned by /tools/{name}/execute.
type executeResponse struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ---------- HTTP Handler ----------

// HTTPHandler returns an http.Handler that exposes the registry as a JSON API.
//
// Routes:
//
//	GET  /tools                 -> list tools
//	POST /tools/{name}/execute  -> execute a tool
func HTTPHandler(reg *Registry) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/tools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		all := reg.All()
		infos := make([]toolInfo, len(all))
		for i, t := range all {
			infos[i] = toolInfo{Name: t.Name(), Description: t.Description()}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(infos)
	})

	// Pattern: /tools/<name>/execute
	mux.HandleFunc("/tools/", func(w http.ResponseWriter, r *http.Request) {
		// Parse the tool name from the path.
		// Expected: /tools/{name}/execute
		path := strings.TrimPrefix(r.URL.Path, "/tools/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) != 2 || parts[1] != "execute" {
			http.Error(w, "not found -- use /tools/{name}/execute", http.StatusNotFound)
			return
		}

		toolName := parts[0]

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tool := reg.Get(toolName)
		if tool == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(executeResponse{
				Error: fmt.Sprintf("tool %q not found", toolName),
			})
			return
		}

		var req executeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(executeResponse{
				Error: fmt.Sprintf("invalid request body: %v", err),
			})
			return
		}

		result, err := tool.Execute(r.Context(), req.Args)

		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(executeResponse{Error: err.Error()})
			return
		}
		json.NewEncoder(w).Encode(executeResponse{Result: result})
	})

	return mux
}

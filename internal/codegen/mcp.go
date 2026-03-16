package codegen

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// MCPServerConfig describes an MCP server integration.
type MCPServerConfig struct {
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	TransportType string   `json:"transport_type"`
	APIKey        string   `json:"api_key,omitempty"`
	Tools         []string `json:"tools,omitempty"`
	Enabled       bool     `json:"enabled"`
}

// MCPToolCall captures a tool execution made through an MCP server.
type MCPToolCall struct {
	ServerName string         `json:"server_name"`
	ToolName   string         `json:"tool_name"`
	Arguments  map[string]any `json:"arguments,omitempty"`
	Result     string         `json:"result,omitempty"`
	Duration   time.Duration  `json:"duration"`
	Error      string         `json:"error,omitempty"`
}

// MCPRegistry stores MCP server configurations.
type MCPRegistry struct {
	mu      sync.RWMutex
	servers map[string]MCPServerConfig
}

// NewMCPRegistry creates a new MCP server registry.
func NewMCPRegistry() *MCPRegistry {
	return &MCPRegistry{servers: make(map[string]MCPServerConfig)}
}

// Register validates and stores an MCP server configuration.
func (r *MCPRegistry) Register(config MCPServerConfig) error {
	if r == nil {
		return ErrConfigInvalid
	}
	if strings.TrimSpace(config.Name) == "" || strings.TrimSpace(config.URL) == "" {
		return ErrConfigInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.servers == nil {
		r.servers = make(map[string]MCPServerConfig)
	}

	r.servers[config.Name] = config
	return nil
}

// Get returns a server config by name.
func (r *MCPRegistry) Get(name string) (MCPServerConfig, bool) {
	if r == nil {
		return MCPServerConfig{}, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	config, ok := r.servers[name]
	return config, ok
}

// List returns all registered servers sorted by name.
func (r *MCPRegistry) List() []MCPServerConfig {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	out := make([]MCPServerConfig, 0, len(r.servers))
	for _, config := range r.servers {
		out = append(out, config)
	}
	r.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out
}

// Remove deletes a server config by name.
func (r *MCPRegistry) Remove(name string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	delete(r.servers, name)
	r.mu.Unlock()
}

// EnabledServers returns only enabled server configs sorted by name.
func (r *MCPRegistry) EnabledServers() []MCPServerConfig {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	out := make([]MCPServerConfig, 0, len(r.servers))
	for _, config := range r.servers {
		if config.Enabled {
			out = append(out, config)
		}
	}
	r.mu.RUnlock()

	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out
}

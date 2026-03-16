package codegen

import (
	"errors"
	"testing"
)

func TestMCPRegistry_RegisterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config MCPServerConfig
		wantErr error
	}{
		{
			name:    "missing name",
			config:  MCPServerConfig{URL: "https://example.com"},
			wantErr: ErrConfigInvalid,
		},
		{
			name:    "missing url",
			config:  MCPServerConfig{Name: "server-a"},
			wantErr: ErrConfigInvalid,
		},
		{
			name:    "valid config",
			config:  MCPServerConfig{Name: "server-a", URL: "https://example.com", Enabled: true},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := NewMCPRegistry()
			err := registry.Register(tc.config)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestMCPRegistry_GetListRemoveAndSortedList(t *testing.T) {
	t.Parallel()

	registry := NewMCPRegistry()
	servers := []MCPServerConfig{
		{Name: "zeta", URL: "https://zeta", Enabled: false},
		{Name: "alpha", URL: "https://alpha", Enabled: true},
		{Name: "beta", URL: "https://beta", Enabled: true},
	}

	for _, server := range servers {
		if err := registry.Register(server); err != nil {
			t.Fatalf("register(%s) failed: %v", server.Name, err)
		}
	}

	got, ok := registry.Get("beta")
	if !ok {
		t.Fatal("expected to get beta server")
	}
	if got.URL != "https://beta" {
		t.Fatalf("expected beta URL https://beta, got %s", got.URL)
	}

	listed := registry.List()
	if len(listed) != 3 {
		t.Fatalf("expected 3 servers from list, got %d", len(listed))
	}

	wantOrder := []string{"alpha", "beta", "zeta"}
	for i, want := range wantOrder {
		if listed[i].Name != want {
			t.Fatalf("expected list[%d] = %s, got %s", i, want, listed[i].Name)
		}
	}

	registry.Remove("beta")
	if _, ok := registry.Get("beta"); ok {
		t.Fatal("expected beta to be removed")
	}

	listed = registry.List()
	if len(listed) != 2 {
		t.Fatalf("expected 2 servers after removal, got %d", len(listed))
	}
}

func TestMCPRegistry_EnabledServersFiltersAndSorted(t *testing.T) {
	t.Parallel()

	registry := NewMCPRegistry()
	servers := []MCPServerConfig{
		{Name: "charlie", URL: "https://charlie", Enabled: true},
		{Name: "alpha", URL: "https://alpha", Enabled: true},
		{Name: "bravo", URL: "https://bravo", Enabled: false},
	}

	for _, server := range servers {
		if err := registry.Register(server); err != nil {
			t.Fatalf("register(%s) failed: %v", server.Name, err)
		}
	}

	enabled := registry.EnabledServers()
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled servers, got %d", len(enabled))
	}

	if enabled[0].Name != "alpha" || enabled[1].Name != "charlie" {
		t.Fatalf("expected enabled servers sorted by name [alpha charlie], got [%s %s]", enabled[0].Name, enabled[1].Name)
	}
}

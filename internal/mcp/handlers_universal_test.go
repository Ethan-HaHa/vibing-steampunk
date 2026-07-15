package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// resultText extracts the single text payload of a CallToolResult for assertions.
func resultText(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if r == nil {
		t.Fatal("result is nil")
	}
	if len(r.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(r.Content))
	}
	tc, ok := r.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", r.Content[0])
	}
	return tc.Text
}

func newTestServer() *Server {
	return NewServer(&Config{
		BaseURL:  "https://sap.example.invalid:44300",
		Username: "testuser",
		Password: "testpass",
		Client:   "001",
		Language: "EN",
	})
}

// TestMissingParamSentinel verifies that routes owning a unique action return a
// targeted "missing required parameter" hint (via *missingParamError) instead of
// the generic "No handler found" when a required parameter is absent. These cases
// resolve at the route layer and never reach the SAP client.
func TestMissingParamSentinel(t *testing.T) {
	srv := newTestServer()

	tests := []struct {
		name       string
		args       map[string]any
		wantSubstr string
	}{
		{
			name:       "grep without scope returns sentinel",
			args:       map[string]any{"action": "grep", "params": map[string]any{"pattern": "SELECT"}},
			wantSubstr: "missing a required parameter",
		},
		{
			name:       "grep with nothing names grep",
			args:       map[string]any{"action": "grep"},
			wantSubstr: `"grep"`,
		},
		{
			name:       "search without query names search",
			args:       map[string]any{"action": "search"},
			wantSubstr: `"search"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := srv.handleUniversalTool(context.Background(), newRequest(tc.args))
			if err != nil {
				t.Fatalf("handleUniversalTool returned error: %v", err)
			}
			text := resultText(t, res)
			if !strings.Contains(text, tc.wantSubstr) {
				t.Errorf("result text = %q, want substring %q", text, tc.wantSubstr)
			}
			if strings.Contains(text, "No handler found") {
				t.Errorf("expected targeted missing-param hint, got generic unhandled message: %s", text)
			}
		})
	}
}

// TestUnhandledUnknownAction verifies the generic fallback still fires for an
// action no route recognizes, and is not confused with a missing parameter.
func TestUnhandledUnknownAction(t *testing.T) {
	srv := newTestServer()
	res, err := srv.handleUniversalTool(context.Background(), newRequest(map[string]any{"action": "totallybogus"}))
	if err != nil {
		t.Fatalf("handleUniversalTool returned error: %v", err)
	}
	text := resultText(t, res)
	if !strings.Contains(text, "No handler found") {
		t.Errorf("expected generic unhandled message, got: %s", text)
	}
	if strings.Contains(text, "missing a required parameter") {
		t.Errorf("unknown action must not report a missing-param hint: %s", text)
	}
}

// TestQuerySQLMissingSqlQuery covers a shared action: query target="SQL" without
// params.sql_query falls through every route to the unhandled fallback, which now
// carries a query-specific hint naming the missing parameter (not the sentinel).
func TestQuerySQLMissingSqlQuery(t *testing.T) {
	srv := newTestServer()
	res, err := srv.handleUniversalTool(context.Background(), newRequest(map[string]any{
		"action": "query",
		"target": "SQL",
	}))
	if err != nil {
		t.Fatalf("handleUniversalTool returned error: %v", err)
	}
	text := resultText(t, res)
	if !strings.Contains(text, "sql_query") {
		t.Errorf("expected query hint to name params.sql_query, got: %s", text)
	}
	if strings.Contains(text, "missing a required parameter") {
		t.Errorf("shared action query must use the fallback hint, not the sentinel: %s", text)
	}
}

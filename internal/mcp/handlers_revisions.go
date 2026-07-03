// Package mcp provides the MCP server implementation for ABAP ADT tools.
// handlers_revisions.go contains handlers for object version history operations.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// --- Version History (Revision) Handlers ---

func (s *Server) handleGetRevisions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, _ := request.GetArguments()["type"].(string)
	name, _ := request.GetArguments()["name"].(string)

	if objectType == "" || name == "" {
		return newToolResultError("type and name are required"), nil
	}

	opts := &adt.GetSourceOptions{}
	if include, ok := request.GetArguments()["include"].(string); ok && include != "" {
		opts.Include = include
	}
	if parent, ok := request.GetArguments()["parent"].(string); ok && parent != "" {
		opts.Parent = parent
	}

	revisions, err := s.adtClient.GetRevisions(ctx, objectType, name, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetRevisions failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(revisions, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleGetRevisionSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	versionURI, ok := request.GetArguments()["version_uri"].(string)
	if !ok || versionURI == "" {
		return newToolResultError("version_uri is required (from GetRevisions output)"), nil
	}

	source, err := s.adtClient.GetRevisionSource(ctx, versionURI)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetRevisionSource failed: %v", err)), nil
	}

	return mcp.NewToolResultText(source), nil
}

// routeRevisionsAction routes version-history actions in hyperfocused mode.
//
// Actions:
//   - "revisions"        → list version history (requires target)
//   - "revision_source"  → fetch source of a specific version (no target needed; uses params.version_uri)
//   - "compare_versions" → unified diff between two versions or against current (requires target + version1_uri)
//
// "revision_source" intentionally does not require a target — only the version_uri from a prior "revisions" call matters.
func (s *Server) routeRevisionsAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	switch action {
	case "revisions":
		if objectType == "" || objectName == "" {
			return newToolResultError("target is required for action 'revisions': use 'TYPE NAME' (e.g. 'FUNC ZDEMO_FUNC')"), true, nil
		}
		args := map[string]any{
			"type": objectType,
			"name": objectName,
		}
		if v := getStringParam(params, "include"); v != "" {
			args["include"] = v
		}
		if v := getStringParam(params, "parent"); v != "" {
			args["parent"] = v
		}
		return s.callHandler(ctx, s.handleGetRevisions, args)

	case "revision_source":
		versionURI := getStringParam(params, "version_uri")
		if versionURI == "" {
			return newToolResultError("params.version_uri is required for action 'revision_source' (obtain it from action 'revisions')"), true, nil
		}
		args := map[string]any{
			"version_uri": versionURI,
		}
		return s.callHandler(ctx, s.handleGetRevisionSource, args)

	case "compare_versions":
		if objectType == "" || objectName == "" {
			return newToolResultError("target is required for action 'compare_versions': use 'TYPE NAME' (e.g. 'FUNC ZDEMO_FUNC')"), true, nil
		}
		version1 := getStringParam(params, "version1_uri")
		if version1 == "" {
			return newToolResultError("params.version1_uri is required for action 'compare_versions' (obtain it from action 'revisions')"), true, nil
		}
		args := map[string]any{
			"type":         objectType,
			"name":         objectName,
			"version1_uri": version1,
		}
		// version2_uri omitted on purpose when empty — handleCompareVersions defaults to "current".
		if v := getStringParam(params, "version2_uri"); v != "" {
			args["version2_uri"] = v
		}
		if v := getStringParam(params, "include"); v != "" {
			args["include"] = v
		}
		if v := getStringParam(params, "parent"); v != "" {
			args["parent"] = v
		}
		return s.callHandler(ctx, s.handleCompareVersions, args)
	}

	return nil, false, nil
}

func (s *Server) handleCompareVersions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, _ := request.GetArguments()["type"].(string)
	name, _ := request.GetArguments()["name"].(string)
	version1, _ := request.GetArguments()["version1_uri"].(string)
	version2, _ := request.GetArguments()["version2_uri"].(string)

	if objectType == "" || name == "" || version1 == "" {
		return newToolResultError("type, name, and version1_uri are required"), nil
	}
	if version2 == "" {
		version2 = "current"
	}

	opts := &adt.GetSourceOptions{}
	if include, ok := request.GetArguments()["include"].(string); ok && include != "" {
		opts.Include = include
	}
	if parent, ok := request.GetArguments()["parent"].(string); ok && parent != "" {
		opts.Parent = parent
	}

	diff, err := s.adtClient.CompareVersions(ctx, objectType, name, version1, version2, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CompareVersions failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(diff, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

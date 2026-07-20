// Package mcp provides the MCP server implementation for ABAP ADT tools.
// handlers_i18n.go contains handlers for translation/internationalization operations.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// --- i18n Handlers ---

func (s *Server) handleGetObjectTextsInLanguage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	lang, ok := request.GetArguments()["language"].(string)
	if !ok || lang == "" {
		return newToolResultError("language is required"), nil
	}

	content, err := s.adtClient.GetObjectTextsInLanguage(ctx, objectURL, lang)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetObjectTextsInLanguage failed: %v", err)), nil
	}

	return mcp.NewToolResultText(content), nil
}

func (s *Server) handleGetDataElementLabels(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	lang, ok := request.GetArguments()["language"].(string)
	if !ok || lang == "" {
		return newToolResultError("language is required"), nil
	}

	labels, err := s.adtClient.GetDataElementLabels(ctx, name, lang)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetDataElementLabels failed: %v", err)), nil
	}

	jsonBytes, err := json.MarshalIndent(labels, "", "  ")
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleGetMessageClassTexts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	lang, ok := request.GetArguments()["language"].(string)
	if !ok || lang == "" {
		return newToolResultError("language is required"), nil
	}

	texts, err := s.adtClient.GetMessageClassTexts(ctx, name, lang)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetMessageClassTexts failed: %v", err)), nil
	}

	if len(texts) == 0 {
		return mcp.NewToolResultText("No messages found."), nil
	}

	jsonBytes, err := json.MarshalIndent(texts, "", "  ")
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleWriteMessageClassTexts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	lang, ok := request.GetArguments()["language"].(string)
	if !ok || lang == "" {
		return newToolResultError("language is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport, _ := request.GetArguments()["transport"].(string)

	// Parse texts from arguments
	textsRaw, ok := request.GetArguments()["texts"]
	if !ok || textsRaw == nil {
		return newToolResultError("texts is required"), nil
	}

	textsJSON, err := json.Marshal(textsRaw)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to parse texts: %v", err)), nil
	}

	var texts []adt.MessageClassMessage
	if err := json.Unmarshal(textsJSON, &texts); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to parse texts: %v", err)), nil
	}

	err = s.adtClient.WriteMessageClassTexts(ctx, name, lang, texts, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteMessageClassTexts failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Message class %s texts updated successfully in language %s.", name, lang)), nil
}

func (s *Server) handleWriteDataElementLabels(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	lang, ok := request.GetArguments()["language"].(string)
	if !ok || lang == "" {
		return newToolResultError("language is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport, _ := request.GetArguments()["transport"].(string)

	labels := &adt.DataElementLabels{}
	if short, ok := request.GetArguments()["short"].(string); ok {
		labels.Short = short
	}
	if medium, ok := request.GetArguments()["medium"].(string); ok {
		labels.Medium = medium
	}
	if long, ok := request.GetArguments()["long"].(string); ok {
		labels.Long = long
	}
	if heading, ok := request.GetArguments()["heading"].(string); ok {
		labels.Heading = heading
	}

	err := s.adtClient.WriteDataElementLabels(ctx, name, lang, labels, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteDataElementLabels failed: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Data element %s labels updated successfully in language %s.", name, lang)), nil
}

func (s *Server) handleGetTextPoolInLanguage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.GetArguments()["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	lang, ok := request.GetArguments()["language"].(string)
	if !ok || lang == "" {
		return newToolResultError("language is required"), nil
	}

	category, _ := request.GetArguments()["category"].(string)

	entries, err := s.adtClient.GetTextPoolInLanguage(ctx, programName, category, lang)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetTextPoolInLanguage failed: %v", err)), nil
	}

	if len(entries) == 0 {
		return mcp.NewToolResultText("No text pool entries found."), nil
	}

	jsonBytes, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleWriteTextPool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	programName, ok := request.GetArguments()["program_name"].(string)
	if !ok || programName == "" {
		return newToolResultError("program_name is required"), nil
	}

	category, ok := request.GetArguments()["category"].(string)
	if !ok || category == "" {
		return newToolResultError("category is required (symbols, selections, or headings)"), nil
	}

	entriesRaw, ok := request.GetArguments()["entries"]
	if !ok || entriesRaw == nil {
		return newToolResultError("entries is required"), nil
	}

	entriesJSON, err := json.Marshal(entriesRaw)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to parse entries: %v", err)), nil
	}

	var entries []adt.TextElement
	if err := json.Unmarshal(entriesJSON, &entries); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to parse entries: %v", err)), nil
	}

	lang, _ := request.GetArguments()["language"].(string)
	transport, _ := request.GetArguments()["transport"].(string)

	result, err := s.adtClient.WriteTextPool(ctx, programName, category, entries, lang, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("WriteTextPool failed: %v", err)), nil
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (s *Server) handleCompareObjectLanguages(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	sourceLang, ok := request.GetArguments()["source_language"].(string)
	if !ok || sourceLang == "" {
		return newToolResultError("source_language is required"), nil
	}

	targetLang, ok := request.GetArguments()["target_language"].(string)
	if !ok || targetLang == "" {
		return newToolResultError("target_language is required"), nil
	}

	comparison, err := s.adtClient.CompareObjectLanguages(ctx, objectURL, sourceLang, targetLang)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CompareObjectLanguages failed: %v", err)), nil
	}

	jsonBytes, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to format result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

// routeI18nAction exposes the translation/text-pool handlers through the universal
// SAP tool. In hyperfocused mode only the single SAP tool is registered, and its
// action route table otherwise has no i18n entry, so these operations would be
// unreachable. Action names are the tool names lowercased.
func (s *Server) routeI18nAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	switch action {
	case "getobjecttexts":
		return s.callHandler(ctx, s.handleGetObjectTextsInLanguage, params)
	case "getdataelementlabels":
		return s.callHandler(ctx, s.handleGetDataElementLabels, params)
	case "getmessageclasstexts":
		return s.callHandler(ctx, s.handleGetMessageClassTexts, params)
	case "writemessageclasstexts":
		return s.callHandler(ctx, s.handleWriteMessageClassTexts, params)
	case "writedataelementlabels":
		return s.callHandler(ctx, s.handleWriteDataElementLabels, params)
	case "gettextpool":
		return s.callHandler(ctx, s.handleGetTextPoolInLanguage, params)
	case "writetextpool":
		return s.callHandler(ctx, s.handleWriteTextPool, params)
	case "comparelanguages":
		return s.callHandler(ctx, s.handleCompareObjectLanguages, params)
	}
	return nil, false, nil
}

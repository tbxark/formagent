package patch

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/structured"
)

const (
	updateFormToolName        = "update_form"
	updateFormToolDescription = "Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user."
)

type updateFormInput struct {
	Operations []Operation `json:"operations" jsonschema:"description=Array of RFC6902 JSON Patch operations to update the form"`
}

type ToolBasedPatchGenerator[T any] struct {
	chain *structured.Chain[Request[T], updateFormInput]
}

// NewToolBasedPatchGenerator 创建基于工具调用的补丁生成器
func NewToolBasedPatchGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedPatchGenerator[T], error) {
	chain, err := structured.NewChain[Request[T], updateFormInput](
		chatModel,
		buildPatchPrompt[T],
		updateFormToolName,
		updateFormToolDescription,
	)
	if err != nil {
		return nil, err
	}
	return &ToolBasedPatchGenerator[T]{chain: chain}, nil
}

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req Request[T]) (UpdateFormArgs, error) {
	slog.Debug("generate patch request", "allowed_paths", len(req.AllowedPaths), "missing_fields", len(req.MissingFields), "has_guidance", len(req.FieldGuidance) > 0, "input_len", len(req.UserInput))
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		slog.Error("generate patch model call failed", "err", err)
		return UpdateFormArgs{}, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil {
		slog.Warn("generate patch returned nil result")
		return UpdateFormArgs{Ops: []Operation{}}, nil
	}
	slog.Debug("generate patch parsed", "operations", len(result.Operations))

	allowedMap := make(map[string]bool)
	for _, path := range req.AllowedPaths {
		allowedMap[path] = true
	}
	if err := ValidatePatchOperations(result.Operations, allowedMap); err != nil {
		slog.Error("generate patch validation failed", "err", err)
		return UpdateFormArgs{}, fmt.Errorf("generated patches failed validation: %w", err)
	}

	return UpdateFormArgs{Ops: result.Operations}, nil
}

func buildPatchPrompt[T any](ctx context.Context, req Request[T]) ([]*schema.Message, error) {
	stateJSON, err := json.Marshal(req.CurrentState)
	if err != nil {
		return nil, fmt.Errorf("marshal form state: %w", err)
	}
	systemPrompt := fmt.Sprintf("You are a form assistant. Analyze user input and call %s to generate RFC6902 JSON Patch operations. Rules: only use explicit user info; use replace for updates and add for new fields; only use allowed paths; if nothing to extract, return empty operations.", updateFormToolName)

	sections := []string{
		fmt.Sprintf("Form state JSON: %s", string(stateJSON)),
		fmt.Sprintf("Allowed paths:\n%s", formatAllowedPaths(req.AllowedPaths)),
	}
	if s := formatMissingFieldsSection(req.MissingFields); s != "" {
		sections = append(sections, s)
	}
	if s := formatFieldGuidanceSection(req.FieldGuidance); s != "" {
		sections = append(sections, s)
	}
	if req.UserInput != "" {
		sections = append(sections, fmt.Sprintf("User input: %s", req.UserInput))
	}
	userPrompt := strings.Join(sections, "\n\n")

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}, nil
}

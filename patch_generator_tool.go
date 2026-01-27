package formagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

const (
	updateFormToolName        = "update_form"
	updateFormToolDescription = "Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user."
)

type updateFormInput struct {
	Operations []PatchOperation `json:"operations" jsonschema:"description=Array of RFC6902 JSON Patch operations to update the form"`
}

type updateFormOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type ToolBasedPatchGenerator[T any] struct {
	chatModel model.ToolCallingChatModel
}

// NewToolBasedPatchGenerator 创建基于工具调用的补丁生成器
func NewToolBasedPatchGenerator[T any](ctx context.Context, chatModel model.ToolCallingChatModel) (*ToolBasedPatchGenerator[T], error) {
	toolFunc := func(ctx context.Context, input *updateFormInput) (*updateFormOutput, error) {
		return &updateFormOutput{
			Success: true,
			Message: fmt.Sprintf("Generated %d patch operations", len(input.Operations)),
		}, nil
	}
	updateTool, err := utils.InferTool(
		updateFormToolName,
		updateFormToolDescription,
		toolFunc,
	)
	if err != nil {
		return nil, err
	}
	toolInfo, err := updateTool.Info(ctx)
	if err != nil {
		return nil, err
	}
	modelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return nil, err
	}
	return &ToolBasedPatchGenerator[T]{chatModel: modelWithTools}, nil
}

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req PatchRequest[T]) (UpdateFormArgs, error) {
	stateJSON, _ := json.MarshalIndent(req.CurrentState, "", "  ")
	slog.Debug("generate patch request", "allowed_paths", len(req.AllowedPaths), "missing_fields", len(req.MissingFields), "has_guidance", len(req.FieldGuidance) > 0, "input_len", len(req.UserInput))

	systemPrompt := fmt.Sprintf(`You are a form-filling assistant. Analyze user input and call the %s tool to generate RFC6902 JSON Patch operations.

Rules:
- Only extract information explicitly provided by the user
- Use "replace" for updating existing fields, "add" for new fields
- Only use paths from the allowed paths list
- If no information to extract, call the tool with an empty operations array`, updateFormToolName)

	userPrompt := fmt.Sprintf(`Current form state:
%s

Allowed paths (you can only modify these):
%s

%s

%s

User input: %s

Call the %s tool to generate patch operations.`,
		string(stateJSON),
		formatAllowedPaths(req.AllowedPaths),
		formatMissingFieldsSection(req.MissingFields),
		formatFieldGuidanceSection(req.FieldGuidance),
		req.UserInput,
		updateFormToolName,
	)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	resp, err := g.chatModel.Generate(ctx, messages)
	if err != nil {
		slog.Error("generate patch model call failed", "err", err)
		return UpdateFormArgs{}, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		slog.Warn("generate patch tool call missing")
		return UpdateFormArgs{Ops: []PatchOperation{}}, nil
	}

	var args string
	for _, tc := range resp.ToolCalls {
		if tc.Function.Name == updateFormToolName {
			args = tc.Function.Arguments
			break
		}
	}
	if args == "" {
		slog.Warn("generate patch tool call not found", "tool", updateFormToolName)
		return UpdateFormArgs{Ops: []PatchOperation{}}, nil
	}

	var input updateFormInput
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		slog.Error("generate patch tool arguments unmarshal failed", "err", err)
		return UpdateFormArgs{}, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	slog.Debug("generate patch parsed", "operations", len(input.Operations))

	allowedMap := make(map[string]bool)
	for _, path := range req.AllowedPaths {
		allowedMap[path] = true
	}
	if err := ValidatePatchOperations(input.Operations, allowedMap); err != nil {
		slog.Error("generate patch validation failed", "err", err)
		return UpdateFormArgs{}, fmt.Errorf("generated patches failed validation: %w", err)
	}

	return UpdateFormArgs{Ops: input.Operations}, nil
}

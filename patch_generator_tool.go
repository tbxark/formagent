package formagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/tbxark/formagent/structuredoutput"
)

const (
	updateFormToolName        = "update_form"
	updateFormToolDescription = "Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user."
)

type updateFormInput struct {
	Operations []PatchOperation `json:"operations" jsonschema:"description=Array of RFC6902 JSON Patch operations to update the form"`
}

type ToolBasedPatchGenerator[T any] struct {
	chain *structuredoutput.Chain[PatchRequest[T], updateFormInput]
}

// NewToolBasedPatchGenerator 创建基于工具调用的补丁生成器
func NewToolBasedPatchGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedPatchGenerator[T], error) {
	chain, err := structuredoutput.NewChain[PatchRequest[T], updateFormInput](
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

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req PatchRequest[T]) (UpdateFormArgs, error) {
	slog.Debug("generate patch request", "allowed_paths", len(req.AllowedPaths), "missing_fields", len(req.MissingFields), "has_guidance", len(req.FieldGuidance) > 0, "input_len", len(req.UserInput))
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		slog.Error("generate patch model call failed", "err", err)
		return UpdateFormArgs{}, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil {
		slog.Warn("generate patch returned nil result")
		return UpdateFormArgs{Ops: []PatchOperation{}}, nil
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

func buildPatchPrompt[T any](ctx context.Context, req PatchRequest[T]) ([]*schema.Message, error) {
	stateJSON, _ := json.MarshalIndent(req.CurrentState, "", "  ")
	systemPrompt := fmt.Sprintf(`You are a form-filling assistant.
Analyze user input and call the %s tool to generate RFC6902 JSON Patch operations.

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

User input: %s`,
		string(stateJSON),
		formatAllowedPaths(req.AllowedPaths),
		formatMissingFieldsSection(req.MissingFields),
		formatFieldGuidanceSection(req.FieldGuidance),
		req.UserInput,
	)

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}, nil
}

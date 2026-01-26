package formagent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type UpdateFormInput struct {
	Operations []PatchOperation `json:"operations" jsonschema:"description=Array of RFC6902 JSON Patch operations to update the form"`
}

type UpdateFormOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

func createUpdateFormTool() (tool.InvokableTool, error) {
	toolFunc := func(ctx context.Context, input *UpdateFormInput) (*UpdateFormOutput, error) {
		return &UpdateFormOutput{
			Success: true,
			Message: fmt.Sprintf("Generated %d patch operations", len(input.Operations)),
		}, nil
	}

	return utils.InferTool(
		"update_form",
		"Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user.",
		toolFunc,
	)
}

type ToolBasedPatchGenerator[T any] struct {
	chatModel model.ToolCallingChatModel
}

// NewToolBasedPatchGenerator 创建基于工具调用的补丁生成器
func NewToolBasedPatchGenerator[T any](chatModel model.ToolCallingChatModel) *ToolBasedPatchGenerator[T] {
	return &ToolBasedPatchGenerator[T]{chatModel: chatModel}
}

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req PatchRequest[T]) (UpdateFormArgs, error) {
	updateTool, err := createUpdateFormTool()
	if err != nil {
		return UpdateFormArgs{}, fmt.Errorf("failed to create tool: %w", err)
	}

	toolInfo, err := getToolInfo(ctx, updateTool)
	if err != nil {
		return UpdateFormArgs{}, fmt.Errorf("failed to get tool info: %w", err)
	}

	modelWithTools, err := g.chatModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return UpdateFormArgs{}, fmt.Errorf("failed to bind tools: %w", err)
	}

	stateJSON, _ := json.MarshalIndent(req.CurrentState, "", "  ")

	systemPrompt := `You are a form-filling assistant. Analyze user input and call the update_form tool to generate RFC6902 JSON Patch operations.

Rules:
- Only extract information explicitly provided by the user
- Use "replace" for updating existing fields, "add" for new fields
- Only use paths from the allowed paths list
- If no information to extract, call the tool with an empty operations array`

	userPrompt := fmt.Sprintf(`Current form state:
%s

Allowed paths (you can only modify these):
%s

%s

%s

User input: %s

Call the update_form tool to generate patch operations.`,
		string(stateJSON),
		formatAllowedPaths(req.AllowedPaths),
		formatMissingFieldsSection(req.MissingFields),
		formatFieldGuidanceSection(req.FieldGuidance),
		req.UserInput,
	)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	resp, err := modelWithTools.Generate(ctx, messages)
	if err != nil {
		return UpdateFormArgs{}, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		return UpdateFormArgs{Ops: []PatchOperation{}}, nil
	}

	toolCall := resp.ToolCalls[0]
	var input UpdateFormInput
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
		return UpdateFormArgs{}, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	allowedMap := make(map[string]bool)
	for _, path := range req.AllowedPaths {
		allowedMap[path] = true
	}
	if err := ValidatePatchOperations(input.Operations, allowedMap); err != nil {
		return UpdateFormArgs{}, fmt.Errorf("generated patches failed validation: %w", err)
	}

	return UpdateFormArgs{Ops: input.Operations}, nil
}

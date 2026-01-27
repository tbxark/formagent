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
	generateResponseToolName        = "generate_response"
	generateResponseToolDescription = "Generate a natural conversational response to guide the user through form completion. Keep responses concise and helpful."
)

type generateResponseInput struct {
	Message         string `json:"message" jsonschema:"required,description=Natural conversational response to the user"`
	SuggestedAction string `json:"suggested_action" jsonschema:"description=Brief description of what the user should do next"`
}

type generateResponseOutput struct {
	Success bool `json:"success"`
}

type ToolBasedDialogueGenerator[T any] struct {
	chatModel model.ToolCallingChatModel
}

// NewToolBasedDialogueGenerator 创建基于工具调用的对话生成器
func NewToolBasedDialogueGenerator[T any](ctx context.Context, chatModel model.ToolCallingChatModel) (*ToolBasedDialogueGenerator[T], error) {
	toolFunc := func(ctx context.Context, input *generateResponseInput) (*generateResponseOutput, error) {
		return &generateResponseOutput{Success: true}, nil
	}
	responseTool, err := utils.InferTool(
		generateResponseToolName,
		generateResponseToolDescription,
		toolFunc,
	)
	if err != nil {
		return nil, err
	}
	toolInfo, err := responseTool.Info(ctx)
	if err != nil {
		return nil, err
	}
	modelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return nil, err
	}
	return &ToolBasedDialogueGenerator[T]{chatModel: modelWithTools}, nil
}

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req DialogueRequest[T]) (*NextTurnPlan, error) {
	stateJSON, _ := json.MarshalIndent(req.CurrentState, "", "  ")
	slog.Debug("generate dialogue request", "phase", req.Phase, "has_input", req.LastUserInput != "", "missing_fields", len(req.MissingFields), "validation_errors", len(req.ValidationErrors))

	systemPrompt := `You are a helpful form-filling assistant. Call the generate_response tool to create natural conversational responses.

Guidelines:
- Keep responses concise but informative
- Use natural language, not technical jargon
- Support both Chinese and English (match the user's language)
- Focus on guiding the user through form completion`

	userPrompt := fmt.Sprintf(`Phase: %s

Form state:
%s

%s

%s

%s

Call the generate_response tool to create a response.`,
		string(req.Phase),
		string(stateJSON),
		formatUserInputSection(req.LastUserInput, req.PatchApplied),
		formatMissingFieldsSectionForDialogue(req.MissingFields, req.Phase),
		formatValidationErrorsSection(req.ValidationErrors),
	)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	resp, err := g.chatModel.Generate(ctx, messages)
	if err != nil {
		slog.Error("generate dialogue model call failed", "err", err)
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		return nil, fmt.Errorf("LLM call failed: no tool calls found")
	}

	toolCall := resp.ToolCalls[0]
	var input generateResponseInput
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
		slog.Error("generate dialogue tool arguments unmarshal failed", "err", err)
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	if input.Message == "" {
		return nil, fmt.Errorf("LLM call failed: message is empty")
	}

	slog.Debug("generate dialogue parsed", "message_len", len(input.Message), "has_suggested_action", input.SuggestedAction != "")
	return &NextTurnPlan{Message: input.Message, SuggestedAction: input.SuggestedAction}, nil
}

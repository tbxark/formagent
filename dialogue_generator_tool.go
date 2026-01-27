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
		slog.Warn("generate dialogue tool call missing")
		return g.generateFallbackDialogue(req), nil
	}

	toolCall := resp.ToolCalls[0]
	var input generateResponseInput
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
		slog.Error("generate dialogue tool arguments unmarshal failed", "err", err)
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	if input.Message == "" {
		slog.Warn("generate dialogue tool message empty")
		return g.generateFallbackDialogue(req), nil
	}

	slog.Debug("generate dialogue parsed", "message_len", len(input.Message), "has_suggested_action", input.SuggestedAction != "")
	return &NextTurnPlan{Message: input.Message, SuggestedAction: input.SuggestedAction}, nil
}

func (g *ToolBasedDialogueGenerator[T]) generateFallbackDialogue(req DialogueRequest[T]) *NextTurnPlan {
	var message string
	var action string

	switch req.Phase {
	case PhaseCollecting:
		if len(req.ValidationErrors) > 0 {
			message = "请修正以下错误：\n"
			for _, err := range req.ValidationErrors {
				message += fmt.Sprintf("- %s\n", err.Message)
			}
			action = "修正验证错误"
		} else if len(req.MissingFields) > 0 {
			message = "请提供以下信息：\n"
			for _, field := range req.MissingFields {
				message += fmt.Sprintf("- %s\n", field.DisplayName)
			}
			action = "提供缺失字段"
		} else {
			message = "所有必填字段已完成，请确认信息是否正确。"
			action = "确认信息"
		}

	case PhaseConfirming:
		message = "请确认以上信息是否正确。您可以输入\"确认\"提交，或\"返回\"继续修改。"
		action = "确认或返回"

	case PhaseSubmitted:
		message = "表单已成功提交！"
		action = "完成"

	case PhaseCancelled:
		message = "表单填写已取消。"
		action = "已取消"

	default:
		message = "请继续填写表单。"
		action = "继续"
	}

	return &NextTurnPlan{Message: message, SuggestedAction: action}
}

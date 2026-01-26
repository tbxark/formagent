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

type GenerateResponseInput struct {
	Message         string `json:"message" jsonschema:"required,description=Natural conversational response to the user"`
	SuggestedAction string `json:"suggested_action" jsonschema:"description=Brief description of what the user should do next"`
}

type GenerateResponseOutput struct {
	Success bool `json:"success"`
}

func createGenerateResponseTool() (tool.InvokableTool, error) {
	toolFunc := func(ctx context.Context, input *GenerateResponseInput) (*GenerateResponseOutput, error) {
		return &GenerateResponseOutput{Success: true}, nil
	}

	return utils.InferTool(
		"generate_response",
		"Generate a natural conversational response to guide the user through form completion. Keep responses concise and helpful.",
		toolFunc,
	)
}

type ToolBasedDialogueGenerator[T any] struct {
	chatModel model.ToolCallingChatModel
}

// NewToolBasedDialogueGenerator 创建基于工具调用的对话生成器
func NewToolBasedDialogueGenerator[T any](chatModel model.ToolCallingChatModel) *ToolBasedDialogueGenerator[T] {
	return &ToolBasedDialogueGenerator[T]{chatModel: chatModel}
}

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req DialogueRequest[T]) (*NextTurnPlan, error) {
	responseTool, err := createGenerateResponseTool()
	if err != nil {
		return nil, fmt.Errorf("failed to create tool: %w", err)
	}

	toolInfo, err := getToolInfo(ctx, responseTool)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool info: %w", err)
	}

	modelWithTools, err := g.chatModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return nil, fmt.Errorf("failed to bind tools: %w", err)
	}

	stateJSON, _ := json.MarshalIndent(req.CurrentState, "", "  ")

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

	resp, err := modelWithTools.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.ToolCalls) == 0 {
		return g.generateFallbackDialogue(req), nil
	}

	toolCall := resp.ToolCalls[0]
	var input GenerateResponseInput
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	if input.Message == "" {
		return g.generateFallbackDialogue(req), nil
	}

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

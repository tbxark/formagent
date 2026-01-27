package dialogue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/structured"
)

const (
	generateResponseToolName        = "generate_response"
	generateResponseToolDescription = "Generate a natural conversational response to guide the user through form completion. Keep responses concise and helpful."
)

type generateResponseInput struct {
	Message         string `json:"message" jsonschema:"required,description=Natural conversational response to the user"`
	SuggestedAction string `json:"suggested_action" jsonschema:"description=Brief description of what the user should do next"`
}

type ToolBasedDialogueGenerator[T any] struct {
	chain *structured.Chain[Request[T], generateResponseInput]
}

// NewToolBasedDialogueGenerator 创建基于工具调用的对话生成器
func NewToolBasedDialogueGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedDialogueGenerator[T], error) {
	chain, err := structured.NewChain[Request[T], generateResponseInput](
		chatModel,
		buildDialoguePrompt[T],
		generateResponseToolName,
		generateResponseToolDescription,
	)
	if err != nil {
		return nil, err
	}
	return &ToolBasedDialogueGenerator[T]{chain: chain}, nil
}

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req Request[T]) (*NextTurnPlan, error) {
	slog.Debug("generate dialogue request", "phase", req.Phase, "has_input", req.LastUserInput != "", "missing_fields", len(req.MissingFields), "validation_errors", len(req.ValidationErrors))
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		slog.Error("generate dialogue model call failed", "err", err)
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil || result.Message == "" {
		return nil, fmt.Errorf("LLM call failed: message is empty")
	}

	slog.Debug("generate dialogue parsed", "message_len", len(result.Message), "has_suggested_action", result.SuggestedAction != "")
	return &NextTurnPlan{Message: result.Message, SuggestedAction: result.SuggestedAction}, nil
}

func buildDialoguePrompt[T any](ctx context.Context, req Request[T]) ([]*schema.Message, error) {
	stateJSON, _ := json.MarshalIndent(req.CurrentState, "", "  ")
	systemPrompt := `You are a helpful form-filling assistant.
Keep responses concise and natural. Match the user's language.
Call the generate_response tool with the final response.`

	userPrompt := fmt.Sprintf(`Phase: %s

Form state:
%s

%s

%s

%s`,
		string(req.Phase),
		string(stateJSON),
		FormatUserInputSection(req.LastUserInput, req.PatchApplied),
		FormatMissingFieldsSectionForDialogue(req.MissingFields, req.Phase),
		FormatValidationErrorsSection(req.ValidationErrors),
	)

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}, nil
}

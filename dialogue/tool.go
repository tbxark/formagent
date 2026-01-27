package dialogue

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
	stateJSON, err := json.Marshal(req.CurrentState)
	if err != nil {
		return nil, fmt.Errorf("marshal form state: %w", err)
	}
	systemPrompt := "You are a helpful form assistant. Keep responses concise and natural. Match the user's language. Call the generate_response tool with the final response."

	sections := []string{
		fmt.Sprintf("Phase: %s", req.Phase),
		fmt.Sprintf("Form state JSON: %s", string(stateJSON)),
	}
	if s := formatUserInputSection(req.LastUserInput, req.PatchApplied); s != "" {
		sections = append(sections, s)
	}
	if s := formatMissingFieldsSectionForDialogue(req.MissingFields, req.Phase); s != "" {
		sections = append(sections, s)
	}
	if s := formatValidationErrorsSection(req.ValidationErrors); s != "" {
		sections = append(sections, s)
	}
	userPrompt := strings.Join(sections, "\n\n")

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}, nil
}

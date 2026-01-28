package dialogue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/structured"
)

const (
	generateResponseToolName        = "generate_response"
	generateResponseToolDescription = "Generate a natural conversational response to guide the user through form completion. Keep responses concise and helpful."
)

type ToolBasedDialogueGenerator[T any] struct {
	chain *structured.Chain[*Request[T], NextTurnPlan]
}

func NewToolBasedDialogueGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedDialogueGenerator[T], error) {
	chain, err := structured.NewChain[*Request[T], NextTurnPlan](
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

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *Request[T]) (*NextTurnPlan, error) {
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil || result.Message == "" {
		return nil, fmt.Errorf("LLM call failed: message is empty")
	}

	return result, nil
}

func buildDialoguePrompt[T any](ctx context.Context, req *Request[T]) ([]*schema.Message, error) {
	stateJSON, err := json.Marshal(req.CurrentState)
	if err != nil {
		return nil, fmt.Errorf("marshal form state: %w", err)
	}
	systemPrompt := "You are a helpful form assistant. Keep responses concise and natural. Match the user's language. Call the `generate_response` tool with the final response."

	sections := []string{
		fmt.Sprintf("# Phase:\n %s", req.Phase),
		fmt.Sprintf("# Form state JSON:\n```json\n%s\n```", string(stateJSON)),
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

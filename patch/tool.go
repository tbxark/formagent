package patch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	einoSchema "github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/structured"
)

const (
	updateFormToolName        = "update_form"
	updateFormToolDescription = "Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user."
)

type ToolBasedPatchGenerator[T any] struct {
	chain *structured.Chain[*Request[T], UpdateFormArgs]
}

func NewToolBasedPatchGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedPatchGenerator[T], error) {
	chain, err := structured.NewChain[*Request[T], UpdateFormArgs](
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

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req *Request[T]) (*UpdateFormArgs, error) {
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	return result, nil
}

func buildPatchPrompt[T any](ctx context.Context, req *Request[T]) ([]*einoSchema.Message, error) {
	stateJSON, err := json.Marshal(req.CurrentState)
	if err != nil {
		return nil, fmt.Errorf("marshal form state: %w", err)
	}

	systemPrompt := fmt.Sprintf("You are a form assistant. Analyze user input and call `%s` to generate RFC6902 JSON Patch operations. Rules: only use explicit user info; use replace for updates and add for new fields; if nothing to extract, return empty operations. The form schema is provided below, and the form is currently being edited, so ignore required fields for now.", updateFormToolName)

	sections := []string{
		fmt.Sprintf("# Form state schema JSON:\n```json\n%s\n```", req.StateSchema),
		fmt.Sprintf("# Current form state JSON:\n```json\n%s\n```", string(stateJSON)),
	}
	if req.AssistantQuestion != "" {
		sections = append(sections, fmt.Sprintf("# Assistant Question:\n%s", req.AssistantQuestion))
	}
	if req.UserAnswer != "" {
		sections = append(sections, fmt.Sprintf("# User Answer:\n%s", req.UserAnswer))
	}
	userPrompt := strings.Join(sections, "\n\n")

	return []*einoSchema.Message{
		einoSchema.SystemMessage(systemPrompt),
		einoSchema.UserMessage(userPrompt),
	}, nil
}

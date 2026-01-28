package patch

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	einoSchema "github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/structured"
	"github.com/tbxark/formagent/types"
)

const (
	updateFormToolName        = "update_form"
	updateFormToolDescription = "Generate RFC6902 JSON Patch operations to update form fields based on user input. Only include operations for information explicitly provided by the user."
)

type ToolBasedPatchGenerator[T any] struct {
	chain *structured.Chain[*types.ToolRequest[T], UpdateFormArgs]
}

func NewToolBasedPatchGenerator[T any](chatModel model.ToolCallingChatModel) (*ToolBasedPatchGenerator[T], error) {
	chain, err := structured.NewChain[*types.ToolRequest[T], UpdateFormArgs](
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

func (g *ToolBasedPatchGenerator[T]) GeneratePatch(ctx context.Context, req *types.ToolRequest[T]) (*UpdateFormArgs, error) {
	result, err := g.chain.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	if result == nil {
		return nil, nil
	}
	return result, nil
}

func buildPatchPrompt[T any](ctx context.Context, req *types.ToolRequest[T]) ([]*einoSchema.Message, error) {
	msg, err := req.ToPromptMessage()
	if err != nil {
		return nil, fmt.Errorf("convert to prompt message failed: %w", err)
	}

	systemPrompt := fmt.Sprintf(`You are a form assistant. Analyze user input and call '%s' to generate RFC6902 JSON Patch operations.

Rules:
- only use explicit user info;
- use replace for updates and add for new fields;
- if nothing to extract, return empty operations.
- The form schema is provided below, and the form is currently being edited, so ignore required fields for now.
`, updateFormToolName)

	return []*einoSchema.Message{
		einoSchema.SystemMessage(systemPrompt),
		einoSchema.UserMessage(msg),
	}, nil
}

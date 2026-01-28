package command

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"

	"github.com/tbxark/formagent/structured"
)

const (
	parseCommandToolName        = "parse_command_intent"
	parseCommandToolDescription = "Analyze user input and determine command intent: cancel, confirm, none."
)

type parseCommandInput struct {
	Intent Command `json:"intent" jsonschema:"required,enum=cancel,enum=confirm,enum=none,description=The user's command intent"`
}

type ToolBasedCommandParser[T any] struct {
	chain *structured.Chain[*types.ToolRequest[T], parseCommandInput]
}

func NewToolBasedCommandParser[T any](chatModel model.ToolCallingChatModel) (*ToolBasedCommandParser[T], error) {
	chain, err := structured.NewChain[*types.ToolRequest[T], parseCommandInput](
		chatModel,
		buildParseCommandPrompt,
		parseCommandToolName,
		parseCommandToolDescription,
	)
	if err != nil {
		return nil, err
	}
	return &ToolBasedCommandParser[T]{chain: chain}, nil
}

func (p *ToolBasedCommandParser[T]) ParseCommand(ctx context.Context, req *types.ToolRequest[T]) (Command, error) {
	result, err := p.chain.Invoke(ctx, req)
	if err != nil {
		return DoNothing, err
	}
	if result == nil || result.Intent == "" {
		return DoNothing, fmt.Errorf("empty intent returned by %s", parseCommandToolName)
	}
	return result.Intent, nil
}

func buildParseCommandPrompt[T any](ctx context.Context, req *types.ToolRequest[T]) ([]*schema.Message, error) {
	message, err := req.ToPromptMessage()
	if err != nil {
		return nil, fmt.Errorf("convert to prompt message failed: %w", err)
	}
	systemPrompt := fmt.Sprintf(`Determine the user's intent for editing the form based on the latest communication between the user and the assistant..
Choose intent: cancel, confirm, edit, do_nothing.
- cancel: user wants to cancel/quit/stop
- confirm: user wants to confirm/submit/done
- edit: user wants to edit/update form fields
- do_nothing: user input is irrelevant to form editing
Call the '%s' tool with the result.`, parseCommandToolName)

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(message),
	}, nil
}

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
	Intent Command `json:"intent" jsonschema:"required,enum=cancel,enum=confirm,enum=edit,enum=do_nothing,description=The user's command intent"`
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

	systemPrompt := fmt.Sprintf(`You are an assistant for a form-filling robot, helping to understand user input in the context of filling out forms.

Analyze the latest communication between the user and the assistant to determine the user's intent regarding form editing.

IMPORTANT: Always combine the assistant's question/prompt with the user's answer to determine the true intent. Do not judge intent solely based on isolated words or phrases. Context is key.

Choose the most appropriate intent from the allowed ones:
- cancel: Only return this if the user explicitly expresses intent to abandon or cancel the current form filling process (e.g., "cancel", "quit", "abandon", "stop filling"). Do not interpret general negations like "no", "don't", "not" as cancel unless they clearly refer to abandoning the process in context.
- confirm: Only return this if the user explicitly expresses intent to confirm and submit the current form (e.g., "confirm", "submit", "yes, proceed", "finalize"). Do not interpret general affirmations like "yes", "ok", "good" as confirm unless they clearly refer to submitting the form in context.
- edit: Return this if the user's input provides information that would change or update form data, such as filling fields, modifying values, or continuing to provide details for the form.
- do_nothing: Return this for purely conversational input, irrelevant chatter, or responses that do not relate to form editing or the current process.

Call the '%s' tool with the result.`, parseCommandToolName)

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(message),
	}, nil
}

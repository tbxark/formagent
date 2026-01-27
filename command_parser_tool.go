package formagent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/tbxark/formagent/structuredoutput"
)

const (
	parseCommandToolName        = "parse_command_intent"
	parseCommandToolDescription = "Analyze user input and determine command intent: cancel, confirm, back, none."
)

type parseCommandInput struct {
	Intent      Command `json:"intent" jsonschema:"required,enum=cancel,enum=confirm,enum=back,enum=none,description=The user's command intent"`
	Explanation string  `json:"explanation,omitempty" jsonschema:"description=Brief reason"`
}

type ToolBasedCommandParser struct {
	chain *structuredoutput.Chain[string, parseCommandInput]
}

func NewToolBasedCommandParser(chatModel model.ToolCallingChatModel) (*ToolBasedCommandParser, error) {
	chain, err := structuredoutput.NewChain[string, parseCommandInput](
		chatModel,
		buildParseCommandPrompt,
		parseCommandToolName,
		parseCommandToolDescription,
	)
	if err != nil {
		return nil, err
	}
	return &ToolBasedCommandParser{chain: chain}, nil
}

func (p *ToolBasedCommandParser) ParseCommand(ctx context.Context, input string) (Command, error) {
	result, err := p.chain.Invoke(ctx, input)
	if err != nil {
		return CommandNone, err
	}
	if result == nil || result.Intent == "" {
		return CommandNone, fmt.Errorf("empty intent returned by %s", parseCommandToolName)
	}
	return result.Intent, nil
}

func buildParseCommandPrompt(ctx context.Context, input string) ([]*schema.Message, error) {
	systemPrompt := fmt.Sprintf(`You classify user intent for form commands.
Choose intent: cancel, confirm, back, none.
- cancel: user wants to cancel/quit/stop
- confirm: user wants to confirm/submit/done
- back: user wants to go back to edit/modify previous content
- none: user is providing information or other actions
Call the %s tool with the result.`, parseCommandToolName)

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(input),
	}, nil
}

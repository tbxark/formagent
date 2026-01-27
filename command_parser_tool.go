package formagent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type ParseCommandInput struct {
	Intent      Command `json:"intent" jsonschema:"required,enum=cancel,enum=confirm,enum=back,enum=none,description=The user's command intent"`
	Explanation string  `json:"explanation,omitempty" jsonschema:"description=Brief reason"`
}

type ParseCommandOutput struct {
	Success bool `json:"success"`
}
type ToolBasedCommandParser struct {
	chatModel model.ToolCallingChatModel
}

func NewToolBasedCommandParser(ctx context.Context, chatModel model.ToolCallingChatModel) (*ToolBasedCommandParser, error) {
	toolFunc := func(ctx context.Context, input *ParseCommandInput) (*ParseCommandOutput, error) {
		return &ParseCommandOutput{Success: true}, nil
	}
	parseTool, err := utils.InferTool(
		"parse_command_intent",
		"Analyze user input and determine command intent: cancel, confirm, back, none.",
		toolFunc,
	)
	if err != nil {
		return nil, err
	}
	toolInfo, err := getToolInfo(ctx, parseTool)
	if err != nil {
		return nil, err
	}
	modelWithTools, err := chatModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return nil, err
	}
	return &ToolBasedCommandParser{chatModel: modelWithTools}, nil
}

func (p *ToolBasedCommandParser) ParseCommand(ctx context.Context, input string) (Command, error) {
	systemPrompt := `You are a command intent recognizer.
You MUST call the tool parse_command_intent and provide JSON arguments that match the tool schema.
Return:
- cancel: user wants to cancel/quit/stop
- confirm: user wants to confirm/submit/done
- back: user explicitly wants to go back to edit/modify previous content
- none: user is providing information or other actions`

	resp, err := p.chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(input),
	})
	if err != nil {
		return CommandNone, err
	}

	// 找到正确的 tool call
	var args string
	for _, tc := range resp.ToolCalls {
		if tc.Function.Name == "parse_command_intent" {
			args = tc.Function.Arguments
			break
		}
	}
	if args == "" {
		return CommandNone, fmt.Errorf("model did not call parse_command_intent tool")
	}

	var cmdInput ParseCommandInput
	err = json.Unmarshal([]byte(args), &cmdInput)
	if err != nil {
		return CommandNone, fmt.Errorf("failed to parse tool arguments: %w", err)
	}
	return cmdInput.Intent, nil
}

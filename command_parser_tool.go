package formagent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type ToolBasedCommandParser struct {
	chatModel model.ToolCallingChatModel
}

// NewToolBasedCommandParser 创建基于工具调用的命令解析器
func NewToolBasedCommandParser(chatModel model.ToolCallingChatModel) *ToolBasedCommandParser {
	return &ToolBasedCommandParser{chatModel: chatModel}
}

type ParseCommandInput struct {
	Intent      string `json:"intent" jsonschema:"required,enum=cancel,enum=confirm,enum=back,enum=none,description=The user's command intent"`
	Confidence  string `json:"confidence" jsonschema:"required,enum=high,enum=medium,enum=low,description=Confidence level of the intent recognition"`
	Explanation string `json:"explanation" jsonschema:"description=Brief explanation of why this intent was chosen"`
}

type ParseCommandOutput struct {
	Success bool `json:"success"`
}

func (p *ToolBasedCommandParser) ParseCommand(ctx context.Context, input string) Command {
	toolFunc := func(ctx context.Context, input *ParseCommandInput) (*ParseCommandOutput, error) {
		return &ParseCommandOutput{Success: true}, nil
	}

	parseTool, err := utils.InferTool(
		"parse_command_intent",
		"Analyze user input and determine their command intent: cancel (stop/quit), confirm (submit/done), back (return to edit), or none (providing information)",
		toolFunc,
	)
	if err != nil {
		return p.fallbackParse(ctx, input)
	}

	toolInfo, err := getToolInfo(ctx, parseTool)
	if err != nil {
		return p.fallbackParse(ctx, input)
	}

	modelWithTools, err := p.chatModel.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return p.fallbackParse(ctx, input)
	}

	systemPrompt := `You are a command intent recognition assistant. Analyze user input and call the parse_command_intent tool.

Rules:
- "cancel": User wants to cancel, quit, or stop the operation
- "confirm": User wants to confirm, submit, or complete the operation
- "back": User wants to return to edit previous content
- "none": User is providing information or doing something else

Important:
- Only return a command when the user clearly expresses that intent
- If user is modifying a specific field (e.g., "change age to 35"), return "none"
- Only return "back" for explicit phrases like "go back to edit" or "return to modify"`

	userPrompt := fmt.Sprintf("User input: %s\n\nCall the parse_command_intent tool to analyze this input.", input)

	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	resp, err := modelWithTools.Generate(ctx, messages)
	if err != nil {
		return p.fallbackParse(ctx, input)
	}

	if len(resp.ToolCalls) == 0 {
		return p.fallbackParse(ctx, input)
	}

	toolCall := resp.ToolCalls[0]
	var cmdInput ParseCommandInput
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &cmdInput); err != nil {
		return p.fallbackParse(ctx, input)
	}

	switch cmdInput.Intent {
	case "cancel":
		return CommandCancel
	case "confirm":
		return CommandConfirm
	case "back":
		return CommandBack
	default:
		return CommandNone
	}
}

func (p *ToolBasedCommandParser) fallbackParse(ctx context.Context, input string) Command {
	defaultParser := NewDefaultCommandParser()
	return defaultParser.ParseCommand(ctx, input)
}

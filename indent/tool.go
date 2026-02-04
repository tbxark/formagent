package indent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"

	"github.com/tbxark/formagent/structured"
)

const (
	parseIntentToolName        = "parse_intent"
	parseIntentToolDescription = "Analyze user input and determine command intent: cancel, confirm, none."
)

// DefaultParseIntentSystemPromptTemplate is the default system prompt template used by
// ToolBasedIntentRecognizer. The template may contain a single "%s" placeholder for the tool name.
const DefaultParseIntentSystemPromptTemplate = `
You are an assistant for a form-filling robot, helping to understand user input in the context of filling out forms.

Analyze the latest communication between the user and the assistant to determine the user's intent regarding form editing.

IMPORTANT: Always combine the assistant's question/prompt with the user's answer to determine the true intent. Do not judge intent solely based on isolated words or phrases. Context is key.

Choose the most appropriate intent from the allowed ones:
- cancel: Only return this if the user explicitly expresses intent to abandon or cancel the current form filling process (e.g., "cancel", "quit", "abandon", "stop filling"). Do not interpret general negations like "no", "don't", "not" as cancel unless they clearly refer to abandoning the process in context.
- confirm: Only return this if the user explicitly expresses intent to confirm and submit the current form (e.g., "confirm", "submit", "yes, proceed", "finalize"). Do not interpret general affirmations like "yes", "ok", "good" as confirm unless they clearly refer to submitting the form in context.
- edit: Return this if the user's input provides information that would change or update form data, such as filling fields, modifying values, or continuing to provide details for the form.
- do_nothing: Return this for purely conversational input, irrelevant chatter, or responses that do not relate to form editing or the current process.

Call the '%s' tool with the result.
`

type PromptBuilder[T any] func(systemPrompt string) func(ctx context.Context, req *types.ToolRequest[T]) ([]*schema.Message, error)

type intentParserOptions[T any] struct {
	systemPromptTemplate string
	promptBuilder        PromptBuilder[T]
}

type ParserOption[T any] func(*intentParserOptions[T])

func WithIntentSystemPromptTemplate[T any](systemPromptTemplate string) ParserOption[T] {
	return func(o *intentParserOptions[T]) {
		o.systemPromptTemplate = systemPromptTemplate
	}
}

func WithIntentPromptBuilder[T any](promptBuilder PromptBuilder[T]) ParserOption[T] {
	return func(o *intentParserOptions[T]) {
		o.promptBuilder = promptBuilder
	}
}

func newIntentRecognizerOptions[T any](opts ...ParserOption[T]) *intentParserOptions[T] {
	opt := intentParserOptions[T]{
		systemPromptTemplate: DefaultParseIntentSystemPromptTemplate,
		promptBuilder: func(systemPrompt string) func(ctx context.Context, req *types.ToolRequest[T]) ([]*schema.Message, error) {
			return func(ctx context.Context, req *types.ToolRequest[T]) ([]*schema.Message, error) {
				message, err := types.FormatToolRequest(req)
				if err != nil {
					return nil, fmt.Errorf("convert to prompt message failed: %w", err)
				}
				return []*schema.Message{
					schema.SystemMessage(systemPrompt),
					schema.UserMessage(message),
				}, nil
			}
		},
	}
	for _, o := range opts {
		o(&opt)
	}
	return &opt
}

type parseCommandInput struct {
	Intent Intent `json:"intent" jsonschema:"required,enum=cancel,enum=confirm,enum=edit,enum=do_nothing,description=The user's command intent"`
}

type ToolBasedIntentRecognizer[T any] struct {
	chain *structured.Chain[*types.ToolRequest[T], parseCommandInput]
}

func NewToolBasedIntentRecognizer[T any](chatModel model.ToolCallingChatModel, opts ...ParserOption[T]) (*ToolBasedIntentRecognizer[T], error) {
	options := newIntentRecognizerOptions[T](opts...)
	chain, err := structured.NewChain[*types.ToolRequest[T], parseCommandInput](
		chatModel,
		options.promptBuilder(fmt.Sprintf(options.systemPromptTemplate, parseIntentToolName)),
		parseIntentToolName,
		parseIntentToolDescription,
	)
	if err != nil {
		return nil, err
	}
	return &ToolBasedIntentRecognizer[T]{chain: chain}, nil
}

func (p *ToolBasedIntentRecognizer[T]) RecognizerIntent(ctx context.Context, req *types.ToolRequest[T]) (Intent, error) {
	result, err := p.chain.Invoke(ctx, req)
	if err != nil {
		return DoNothing, err
	}
	if result == nil || result.Intent == "" {
		return DoNothing, fmt.Errorf("empty intent returned by %s", parseIntentToolName)
	}
	return result.Intent, nil
}

package dialogue

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

// DefaultDialogueSystemPrompt is the default system prompt template used by
// ToolBasedDialogueGenerator. The template may contain a single "%s" placeholder for the language.
const DefaultDialogueSystemPrompt = `

You are a form-completion dialogue assistant. Your task is to guide the user to complete and submit a form based on the provided textual context.

## Context Interpretation Rules
- The input may contain multiple clearly labeled sections, including but not limited to:
- Current date
- Form state (as JSON)
- Form state summary
- Current phase
- Dialogue history
- Missing required fields
- Validation errors
- Section names, field names, and formatting may vary, but their semantic meaning should be inferred from the content.
- Treat the rendered text as the single source of truth about the form’s current status.

## Dialogue Behavior Rules
- Use a natural, conversational tone, as if chatting with a friend.
- Do not expose or reference internal section titles, tables, JSON, pointers, or system formatting in the reply.
- Never repeat raw field pointers or structured metadata verbatim unless necessary for clarity; prefer user-facing field names.

## Form Guidance Rules
- If missing required fields are present, ask for them casually and incrementally; do not request many at once.
- If validation errors are present, gently explain the issue and suggest a correction in simple terms.
- If both missing fields and validation errors exist, prioritize addressing validation errors first.
- Acknowledge correctly completed fields or progress when appropriate.
- If the form is complete and valid, explicitly ask whether the user wants to submit it.
- If the dialogue history indicates the user changed a value, confirm the update and reflect the latest form status.

## Language Constraint
- Always reply in **Simplified Chinese**.

## Output Constraint
- Do not use lists, bullet points, tables, or headings in user-facing responses.
- Keep responses concise, precise, and conversational.
`

type PromptBuilder[T any] func(systemPrompt string) func(ctx context.Context, req *types.ToolRequest[T]) ([]*schema.Message, error)

type dialogueGeneratorOptions[T any] struct {
	systemPrompt  string
	promptBuilder PromptBuilder[T]
}

type GeneratorOption[T any] func(*dialogueGeneratorOptions[T])

func WithDialogueSystemPromptTemplate[T any](systemPromptTemplate string) GeneratorOption[T] {
	return func(o *dialogueGeneratorOptions[T]) {
		o.systemPrompt = systemPromptTemplate
	}
}

func WithDialoguePromptBuilder[T any](promptBuilder PromptBuilder[T]) GeneratorOption[T] {
	return func(o *dialogueGeneratorOptions[T]) {
		o.promptBuilder = promptBuilder
	}
}

func newDialogueGeneratorOptions[T any](opts ...GeneratorOption[T]) dialogueGeneratorOptions[T] {
	opt := dialogueGeneratorOptions[T]{
		systemPrompt: DefaultDialogueSystemPrompt,
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
	return opt
}

type ToolBasedDialogueGenerator[T any] struct {
	promptBuilder func(ctx context.Context, req *types.ToolRequest[T]) ([]*schema.Message, error)
	chatModel     model.ToolCallingChatModel
}

func NewToolBasedDialogueGenerator[T any](chatModel model.ToolCallingChatModel, opts ...GeneratorOption[T]) *ToolBasedDialogueGenerator[T] {
	options := newDialogueGeneratorOptions[T](opts...)
	return &ToolBasedDialogueGenerator[T]{
		promptBuilder: options.promptBuilder(options.systemPrompt),
		chatModel:     chatModel,
	}
}

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *types.ToolRequest[T]) (string, error) {
	messages, err := g.promptBuilder(ctx, req)
	if err != nil {
		return "", fmt.Errorf("build dialogue prompt: %w", err)
	}

	response, err := g.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}
	return response.Content, nil
}

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogueStream(ctx context.Context, req *types.ToolRequest[T]) (*schema.StreamReader[string], error) {
	messages, err := g.promptBuilder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("build dialogue prompt: %w", err)
	}

	stream, err := g.chatModel.Stream(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM stream call failed: %w", err)
	}
	textStream := schema.StreamReaderWithConvert[*schema.Message, string](stream, func(message *schema.Message) (string, error) {
		return message.Content, nil
	})
	return textStream, nil
}

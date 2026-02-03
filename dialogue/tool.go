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
You are a friendly form assistant. Engage in natural, conversational dialogue to guide users through form completion.

Respond as if chatting with a friend:
- If there are missing required fields, casually mention them and ask for the information in a friendly way. Don't list all at once if there are many.
- If there are validation errors, gently point them out and suggest corrections using simple, easy-to-understand language.
- Acknowledge what they've already filled out to make them feel good.
- If all fields are complete and correct, actively ask if they want to submit the form.
- Avoid lists or bullet points; make it feel like a real conversation.
- Reply in **Simplified Chinese**.
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

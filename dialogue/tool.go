package dialogue

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type ToolBasedDialogueGenerator[T any] struct {
	Lang                 string
	systemPrompt         string
	systemPromptTemplate string
	chatModel            model.ToolCallingChatModel
}

// DefaultDialogueSystemPromptTemplate is the default system prompt template used by
// ToolBasedDialogueGenerator. The template may contain a single "%s" placeholder for the language.
const DefaultDialogueSystemPromptTemplate = `You are a friendly form assistant. Engage in natural, conversational dialogue to guide users through form completion.

Respond as if chatting with a friend:
- If there are missing required fields, casually mention them and ask for the information in a friendly way. Don't list all at once if there are many.
- If there are validation errors, gently point them out and suggest corrections using simple, easy-to-understand language.
- Acknowledge what they've already filled out to make them feel good.
- If all fields are complete and correct, actively ask if they want to submit the form.
- Avoid lists or bullet points; make it feel like a real conversation.
- Reply in %s.
`

type dialogueGeneratorOptions struct {
	lang                 string
	systemPrompt         string
	systemPromptTemplate string
}

type GeneratorOption func(*dialogueGeneratorOptions)

// WithDialogueLang sets the language used by the default system prompt template.
func WithDialogueLang(lang string) GeneratorOption {
	return func(o *dialogueGeneratorOptions) {
		o.lang = lang
	}
}

// WithDialogueSystemPrompt overrides the system prompt used by ToolBasedDialogueGenerator.
func WithDialogueSystemPrompt(systemPrompt string) GeneratorOption {
	return func(o *dialogueGeneratorOptions) {
		o.systemPrompt = systemPrompt
	}
}

// WithDialogueSystemPromptTemplate overrides the system prompt template used by ToolBasedDialogueGenerator.
// If the template contains "%s", it will be formatted with the language.
func WithDialogueSystemPromptTemplate(systemPromptTemplate string) GeneratorOption {
	return func(o *dialogueGeneratorOptions) {
		o.systemPromptTemplate = systemPromptTemplate
	}
}

func NewToolBasedDialogueGenerator[T any](chatModel model.ToolCallingChatModel, opts ...GeneratorOption) *ToolBasedDialogueGenerator[T] {
	options := dialogueGeneratorOptions{
		lang:                 "Simplified Chinese",
		systemPromptTemplate: DefaultDialogueSystemPromptTemplate,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.lang == "" {
		options.lang = "Simplified Chinese"
	}
	return &ToolBasedDialogueGenerator[T]{
		Lang:                 options.lang,
		systemPrompt:         options.systemPrompt,
		systemPromptTemplate: options.systemPromptTemplate,
		chatModel:            chatModel,
	}
}

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *types.ToolRequest[T]) (string, error) {
	messages, err := g.buildDialoguePrompt(req)
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
	messages, err := g.buildDialoguePrompt(req)
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

func (g *ToolBasedDialogueGenerator[T]) buildDialoguePrompt(req *types.ToolRequest[T]) ([]*schema.Message, error) {

	message, err := types.FormatToolRequest(req)
	if err != nil {
		return nil, fmt.Errorf("convert to prompt message failed: %w", err)
	}

	systemPrompt := g.systemPrompt
	if systemPrompt == "" {
		tpl := g.systemPromptTemplate
		if tpl == "" {
			tpl = DefaultDialogueSystemPromptTemplate
		}
		if strings.Contains(tpl, "%s") {
			systemPrompt = fmt.Sprintf(tpl, g.Lang)
		} else {
			systemPrompt = tpl
		}
	}

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(message),
	}, nil
}

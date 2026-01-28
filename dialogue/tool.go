package dialogue

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type ToolBasedDialogueGenerator[T any] struct {
	Lang      string
	chatModel model.ToolCallingChatModel
}

func NewToolBasedDialogueGenerator[T any](chatModel model.ToolCallingChatModel) *ToolBasedDialogueGenerator[T] {
	return &ToolBasedDialogueGenerator[T]{
		Lang:      "Simplified Chinese",
		chatModel: chatModel,
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

	message, err := req.ToPromptMessage()
	if err != nil {
		return nil, fmt.Errorf("convert to prompt message failed: %w", err)
	}

	systemPrompt := fmt.Sprintf(`You are a friendly form assistant. Engage in natural, conversational dialogue to guide users through form completion.

Respond as if chatting with a friend:
- If there are missing required fields, casually mention them and ask for the information in a friendly way. Don't list all at once if there are many.
- If there are validation errors, gently point them out and suggest corrections using simple, easy-to-understand language.
- Acknowledge what they've already filled out to make them feel good.
- If all fields are complete and correct, actively ask if they want to submit the form.
- Avoid lists or bullet points; make it feel like a real conversation.
- Reply in %s.
`, g.Lang)

	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(message),
	}, nil
}

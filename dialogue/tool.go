package dialogue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const (
	generateResponseToolName        = "generate_response"
	generateResponseToolDescription = "Generate a natural conversational response to guide the user through form completion. Keep responses concise and helpful."
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

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *Request[T]) (string, error) {
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

func (g *ToolBasedDialogueGenerator[T]) GenerateDialogueStream(ctx context.Context, req *Request[T]) (*schema.StreamReader[string], error) {
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

func (g *ToolBasedDialogueGenerator[T]) buildDialoguePrompt(req *Request[T]) ([]*schema.Message, error) {
	stateJSON, err := json.Marshal(req.CurrentState)
	if err != nil {
		return nil, fmt.Errorf("marshal form state: %w", err)
	}
	systemPrompt := fmt.Sprintf(`You are a friendly form assistant. Engage in natural, conversational dialogue to guide users through form completion.

Respond as if chatting with a friend:
- ask questions casually, acknowledge what they've filled out, and gently prompt for missing information.
- Avoid lists or bullet points; make it feel like a real conversation.
- Reply in %s.
- Always call the '%s' tool with your final response.`, g.Lang, generateResponseToolName)

	sections := []string{
		fmt.Sprintf("# Phase:\n %s", req.Phase),
		fmt.Sprintf("# Form state JSON:\n```json\n%s\n```", string(stateJSON)),
	}
	if s := formatMissingFieldsSectionForDialogue(req.MissingFields); s != "" {
		sections = append(sections, s)
	}
	if s := formatValidationErrorsSection(req.ValidationErrors); s != "" {
		sections = append(sections, s)
	}
	userPrompt := strings.Join(sections, "\n\n")
	return []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}, nil
}

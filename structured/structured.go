package structured

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type PromptBuilder[TInput any] func(ctx context.Context, input TInput) ([]*schema.Message, error)

type Chain[TInput, TOutput any] struct {
	PromptBuilder PromptBuilder[TInput]
	ChatModel     model.ToolCallingChatModel
	ToolInfo      *schema.ToolInfo
}

func NewChain[TInput, TOutput any](
	chatModel model.ToolCallingChatModel,
	promptBuilder PromptBuilder[TInput],
	toolName string,
	toolDesc string,
) (*Chain[TInput, TOutput], error) {

	toolInfo, err := utils.GoStruct2ToolInfo[TOutput](toolName, toolDesc)
	if err != nil {
		return nil, fmt.Errorf("convert tool info failed: %w", err)
	}
	return &Chain[TInput, TOutput]{
		PromptBuilder: promptBuilder,
		ChatModel:     chatModel,
		ToolInfo:      toolInfo,
	}, nil
}

func (s *Chain[TInput, TOutput]) Invoke(ctx context.Context, input TInput) (*TOutput, error) {
	messages, err := s.PromptBuilder(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("build prompt failed: %w", err)
	}

	response, err := s.ChatModel.Generate(ctx, messages,
		model.WithTools([]*schema.ToolInfo{s.ToolInfo}),
		model.WithToolChoice(schema.ToolChoiceForced, s.ToolInfo.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("call model failed: %w", err)
	}
	if len(response.ToolCalls) == 0 {
		return nil, fmt.Errorf("no ToolCall found in model response: %s", response.Content)
	}

	var result TOutput
	if err := sonic.UnmarshalString(response.ToolCalls[0].Function.Arguments, &result); err != nil {
		return nil, fmt.Errorf("parse ToolCall arguments failed: %w", err)
	}

	return &result, nil
}

func (s *Chain[TInput, TOutput]) Stream(ctx context.Context, input TInput) (*schema.StreamReader[*TOutput], error) {
	messages, err := s.PromptBuilder(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("build prompt failed: %w", err)
	}

	streamReader, err := s.ChatModel.Stream(ctx, messages,
		model.WithTools([]*schema.ToolInfo{s.ToolInfo}),
		model.WithToolChoice(schema.ToolChoiceForced, s.ToolInfo.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("call model failed: %w", err)
	}

	outputReader := schema.StreamReaderWithConvert(streamReader, func(msg *schema.Message) (*TOutput, error) {
		if len(msg.ToolCalls) == 0 {
			return nil, fmt.Errorf("no ToolCall found in model response: %s", msg.Content)
		}

		var result TOutput
		if err := sonic.UnmarshalString(msg.ToolCalls[0].Function.Arguments, &result); err != nil {
			return nil, fmt.Errorf("parse ToolCall arguments failed: %w", err)
		}

		return &result, nil
	})

	return outputReader, nil
}

func (s *Chain[TInput, TOutput]) GetToolInfo() *schema.ToolInfo {
	return s.ToolInfo
}

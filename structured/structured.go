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
	promptBuilder PromptBuilder[TInput]
	chatModel     model.ToolCallingChatModel
	toolInfo      *schema.ToolInfo
}

func NewChain[TInput, TOutput any](
	chatModel model.ToolCallingChatModel,
	promptBuilder PromptBuilder[TInput],
	toolName string,
	toolDesc string,
) (*Chain[TInput, TOutput], error) {

	toolInfo, err := utils.GoStruct2ToolInfo[TOutput](toolName, toolDesc)
	if err != nil {
		return nil, fmt.Errorf("创建 ToolInfo 失败: %w", err)
	}

	return &Chain[TInput, TOutput]{
		promptBuilder: promptBuilder,
		chatModel:     chatModel,
		toolInfo:      toolInfo,
	}, nil
}

func (s *Chain[TInput, TOutput]) Invoke(ctx context.Context, input TInput) (*TOutput, error) {
	messages, err := s.promptBuilder(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("构建提示词失败: %w", err)
	}

	response, err := s.chatModel.Generate(ctx, messages,
		model.WithTools([]*schema.ToolInfo{s.toolInfo}),
		model.WithToolChoice(schema.ToolChoiceForced, s.toolInfo.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("调用模型失败: %w", err)
	}

	if len(response.ToolCalls) == 0 {
		return nil, fmt.Errorf("模型未返回 ToolCall，响应内容: %s", response.Content)
	}

	var result TOutput
	if err := sonic.UnmarshalString(response.ToolCalls[0].Function.Arguments, &result); err != nil {
		return nil, fmt.Errorf("解析 ToolCall 参数失败: %w, 参数内容: %s",
			err, response.ToolCalls[0].Function.Arguments)
	}

	return &result, nil
}

func (s *Chain[TInput, TOutput]) Stream(ctx context.Context, input TInput) (*schema.StreamReader[*TOutput], error) {
	messages, err := s.promptBuilder(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("构建提示词失败: %w", err)
	}

	streamReader, err := s.chatModel.Stream(ctx, messages,
		model.WithTools([]*schema.ToolInfo{s.toolInfo}),
		model.WithToolChoice(schema.ToolChoiceForced, s.toolInfo.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("流式调用模型失败: %w", err)
	}

	outputReader := schema.StreamReaderWithConvert(streamReader, func(msg *schema.Message) (*TOutput, error) {
		if len(msg.ToolCalls) == 0 {
			return nil, fmt.Errorf("流式响应中未找到 ToolCall")
		}

		var result TOutput
		if err := sonic.UnmarshalString(msg.ToolCalls[0].Function.Arguments, &result); err != nil {
			return nil, fmt.Errorf("解析 ToolCall 参数失败: %w", err)
		}

		return &result, nil
	})

	return outputReader, nil
}

func (s *Chain[TInput, TOutput]) GetToolInfo() *schema.ToolInfo {
	return s.toolInfo
}

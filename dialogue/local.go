package dialogue

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type LocalDialogueGenerator[T any] struct{}

func (g *LocalDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *Request[T]) (string, error) {
	var message string
	switch req.Phase {
	case types.PhaseCollecting:
		if len(req.ValidationErrors) > 0 {
			message = "请修正以下错误：\n"
			for _, err := range req.ValidationErrors {
				message += fmt.Sprintf("- %s\n", err.Description)
			}
		} else if len(req.MissingFields) > 0 {
			message = "请提供以下信息：\n"
			for _, field := range req.MissingFields {
				message += fmt.Sprintf("- %s\n", field.DisplayName)
			}
		} else {
			message = "所有必填字段已完成，请确认信息是否正确。"
		}

	case types.PhaseConfirming:
		message = "请确认以上信息是否正确。您可以输入\"确认\"提交，或\"返回\"继续修改。"

	case types.PhaseConfirmed:
		message = "表单已成功提交！"

	case types.PhaseCancelled:
		message = "表单填写已取消。"

	default:
		message = "请继续填写表单。"
	}
	return message, nil
}

func (g *LocalDialogueGenerator[T]) GenerateDialogueStream(ctx context.Context, req *Request[T]) (*schema.StreamReader[string], error) {
	message, err := g.GenerateDialogue(ctx, req)
	if err != nil {
		return nil, err
	}
	stream := schema.StreamReaderFromArray([]string{message})
	return stream, nil
}

type FailbackDialogueGenerator[T any] struct {
	generators []Generator[T]
}

func NewFailbackDialogueGenerator[T any](generators ...Generator[T]) *FailbackDialogueGenerator[T] {
	return &FailbackDialogueGenerator[T]{generators: generators}
}

func (g *FailbackDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *Request[T]) (string, error) {
	var lastErr error
	for _, generator := range g.generators {
		plan, err := generator.GenerateDialogue(ctx, req)
		if err == nil {
			return plan, nil
		}
		lastErr = err
	}
	return "", fmt.Errorf("all dialogue generators failed: %w", lastErr)
}

func (g *FailbackDialogueGenerator[T]) GenerateDialogueStream(ctx context.Context, req *Request[T]) (*schema.StreamReader[string], error) {
	var lastErr error
	for _, generator := range g.generators {
		stream, err := generator.GenerateDialogueStream(ctx, req)
		if err == nil {
			return stream, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all dialogue generators failed: %w", lastErr)
}

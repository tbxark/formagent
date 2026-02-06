package dialogue

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type LocalDialogueGenerator[T any] struct {
	MergeAllUnvalidatedFields bool
}

func (g *LocalDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *types.ToolRequest[T]) (string, error) {
	switch req.Phase {
	case types.PhaseCollecting:
		var sb strings.Builder
		if len(req.ValidationErrors) > 0 {
			for _, err := range req.ValidationErrors {
				if len(err.Description) > 0 {
					sb.WriteString(err.Description)
					sb.WriteString("\n")
				} else {
					sb.WriteString(fmt.Sprintf("必须填写%s\n", err.DisplayName))
				}
				if !g.MergeAllUnvalidatedFields {
					return sb.String(), nil
				}
			}
		}
		if len(req.MissingFields) > 0 {
			for _, field := range req.MissingFields {
				if len(field.Description) > 0 {
					sb.WriteString(field.Description)
					sb.WriteString("\n")
				} else {
					sb.WriteString(fmt.Sprintf("%s填写错误\n", field.DisplayName))
				}
				if !g.MergeAllUnvalidatedFields && sb.Len() > 0 {
					return sb.String(), nil
				}
			}
		}
		if sb.Len() == 0 {
			return "请继续填写表单。", nil
		}
		return sb.String(), nil
	case types.PhaseConfirmed:
		return "表单已成功提交！", nil

	case types.PhaseCancelled:
		return "表单填写已取消。", nil

	default:
		return "请继续填写表单。", nil
	}
}

func (g *LocalDialogueGenerator[T]) GenerateDialogueStream(ctx context.Context, req *types.ToolRequest[T]) (*schema.StreamReader[string], error) {
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

func (g *FailbackDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req *types.ToolRequest[T]) (string, error) {
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

func (g *FailbackDialogueGenerator[T]) GenerateDialogueStream(ctx context.Context, req *types.ToolRequest[T]) (*schema.StreamReader[string], error) {
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

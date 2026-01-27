package dialogue

import (
	"context"
	"fmt"

	"github.com/tbxark/formagent/types"
)

type LocalDialogueGenerator[T any] struct{}

func (g *LocalDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req DialogueRequest[T]) (*NextTurnPlan, error) {
	var message string
	var action string
	switch req.Phase {
	case types.PhaseCollecting:
		if len(req.ValidationErrors) > 0 {
			message = "请修正以下错误：\n"
			for _, err := range req.ValidationErrors {
				message += fmt.Sprintf("- %s\n", err.Message)
			}
			action = "修正验证错误"
		} else if len(req.MissingFields) > 0 {
			message = "请提供以下信息：\n"
			for _, field := range req.MissingFields {
				message += fmt.Sprintf("- %s\n", field.DisplayName)
			}
			action = "提供缺失字段"
		} else {
			message = "所有必填字段已完成，请确认信息是否正确。"
			action = "确认信息"
		}

	case types.PhaseConfirming:
		message = "请确认以上信息是否正确。您可以输入\"确认\"提交，或\"返回\"继续修改。"
		action = "确认或返回"

	case types.PhaseSubmitted:
		message = "表单已成功提交！"
		action = "完成"

	case types.PhaseCancelled:
		message = "表单填写已取消。"
		action = "已取消"

	default:
		message = "请继续填写表单。"
		action = "继续"
	}
	return &NextTurnPlan{Message: message, SuggestedAction: action}, nil
}

type FailbackDialogueGenerator[T any] struct {
	generators []DialogueGenerator[T]
}

func NewFailbackDialogueGenerator[T any](generators ...DialogueGenerator[T]) *FailbackDialogueGenerator[T] {
	return &FailbackDialogueGenerator[T]{generators: generators}
}

func (g *FailbackDialogueGenerator[T]) GenerateDialogue(ctx context.Context, req DialogueRequest[T]) (*NextTurnPlan, error) {
	var lastErr error
	for _, generator := range g.generators {
		plan, err := generator.GenerateDialogue(ctx, req)
		if err == nil {
			return plan, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all dialogue generators failed: %w", lastErr)
}

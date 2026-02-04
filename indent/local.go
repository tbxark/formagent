package indent

import (
	"context"
	"strings"

	"github.com/tbxark/formagent/types"
)

type LocalIntentRecognizer[T any] struct {
	CancelKeywords  []string
	ConfirmKeywords []string
}

func NewLocalIntentRecognizer[T any]() *LocalIntentRecognizer[T] {
	return &LocalIntentRecognizer[T]{
		CancelKeywords:  []string{"取消", "cancel", "退出", "quit", "exit", "停止", "stop"},
		ConfirmKeywords: []string{"确认", "confirm", "提交", "submit", "完成", "done", "好的", "ok", "好"},
	}
}

func (p *LocalIntentRecognizer[T]) RecognizerIntent(ctx context.Context, req *types.ToolRequest[T]) (Intent, error) {
	if len(req.Messages) == 0 {
		return DoNothing, nil
	}
	normalized := strings.ToLower(strings.TrimSpace(req.Messages[len(req.Messages)-1].Content))
	for _, keyword := range p.CancelKeywords {
		if normalized == keyword {
			return Cancel, nil
		}
	}
	for _, keyword := range p.ConfirmKeywords {
		if normalized == keyword {
			return Confirm, nil
		}
	}
	return DoNothing, nil
}

type FailbackCommandParser[T any] struct {
	parsers []Recognizer[T]
}

func NewFailbackCommandParser[T any](parsers ...Recognizer[T]) *FailbackCommandParser[T] {
	return &FailbackCommandParser[T]{parsers: parsers}
}

func (p *FailbackCommandParser[T]) RecognizerIntent(ctx context.Context, req *types.ToolRequest[T]) (Intent, error) {
	var lastErr error
	for _, parser := range p.parsers {
		cmd, err := parser.RecognizerIntent(ctx, req)
		if err == nil {
			return cmd, nil
		}
		lastErr = err
	}
	return DoNothing, lastErr
}

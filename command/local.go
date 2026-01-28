package command

import (
	"context"
	"strings"

	"github.com/tbxark/formagent/types"
)

type LocalCommandParser[T any] struct {
	CancelKeywords  []string
	ConfirmKeywords []string
}

func NewLocalCommandParser[T any]() *LocalCommandParser[T] {
	return &LocalCommandParser[T]{
		CancelKeywords:  []string{"取消", "cancel", "退出", "quit", "exit", "停止", "stop"},
		ConfirmKeywords: []string{"确认", "confirm", "提交", "submit", "完成", "done", "好的", "ok", "好"},
	}
}

func (p *LocalCommandParser[T]) ParseCommand(ctx context.Context, req *types.ToolRequest[T]) (Command, error) {
	normalized := strings.ToLower(strings.TrimSpace(req.MessagePair.Answer))
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
	parsers []Parser[T]
}

func NewFailbackCommandParser[T any](parsers ...Parser[T]) *FailbackCommandParser[T] {
	return &FailbackCommandParser[T]{parsers: parsers}
}

func (p *FailbackCommandParser[T]) ParseCommand(ctx context.Context, req *types.ToolRequest[T]) (Command, error) {
	var lastErr error
	for _, parser := range p.parsers {
		cmd, err := parser.ParseCommand(ctx, req)
		if err == nil {
			return cmd, nil
		}
		lastErr = err
	}
	return DoNothing, lastErr
}

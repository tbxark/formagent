package command

import (
	"context"
	"strings"
)

type StaticCommandParser struct {
	CancelKeywords  []string
	ConfirmKeywords []string
	BackKeywords    []string
}

type DefaultCommandParser = StaticCommandParser

// NewDefaultCommandParser 创建默认命令解析器
func NewDefaultCommandParser() *StaticCommandParser {
	return &StaticCommandParser{
		CancelKeywords:  []string{"取消", "cancel", "退出", "quit", "exit", "停止", "stop"},
		ConfirmKeywords: []string{"确认", "confirm", "提交", "submit", "完成", "done", "好的", "ok", "好"},
		BackKeywords:    []string{"返回", "back", "返回修改", "返回编辑"},
	}
}

func (p *StaticCommandParser) ParseCommand(ctx context.Context, input string) (Command, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))

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

	for _, keyword := range p.BackKeywords {
		if normalized == keyword {
			return Back, nil
		}
	}

	return None, nil
}

type FailbackCommandParser struct {
	parsers []Parser
}

func NewFailbackCommandParser(parsers ...Parser) *FailbackCommandParser {
	return &FailbackCommandParser{parsers: parsers}
}

func (p *FailbackCommandParser) ParseCommand(ctx context.Context, input string) (Command, error) {
	var lastErr error
	for _, parser := range p.parsers {
		cmd, err := parser.ParseCommand(ctx, input)
		if err == nil {
			return cmd, nil
		}
		lastErr = err
	}
	return None, lastErr
}

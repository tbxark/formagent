package formagent

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
			return CommandCancel, nil
		}
	}

	for _, keyword := range p.ConfirmKeywords {
		if normalized == keyword {
			return CommandConfirm, nil
		}
	}

	for _, keyword := range p.BackKeywords {
		if normalized == keyword {
			return CommandBack, nil
		}
	}

	return CommandNone, nil
}

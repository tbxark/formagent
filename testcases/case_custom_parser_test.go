package testcases

import (
	"context"
	"testing"

	"github.com/tbxark/formagent"
)

// CustomCommandParser 自定义命令解析器示例
type CustomCommandParser struct {
	// 可以添加自定义配置
}

func (p *CustomCommandParser) ParseCommand(ctx context.Context, input string) formagent.Command {
	// 自定义逻辑：例如支持更多语言或特殊规则
	switch input {
	case "不要了", "算了":
		return formagent.CommandCancel
	case "就这样", "可以了":
		return formagent.CommandConfirm
	case "再看看", "重新来":
		return formagent.CommandBack
	default:
		// 回退到默认解析器
		defaultParser := formagent.NewDefaultCommandParser()
		return defaultParser.ParseCommand(ctx, input)
	}
}

// TestCustomCommandParser 测试自定义命令解析器
func TestCustomCommandParser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// 使用自定义命令解析器
	customParser := &CustomCommandParser{}
	agent := NewTestAgent(t, WithCommandParser(customParser))

	// 填写一些信息
	if _, err := agent.Invoke(ctx, "我叫测试用户"); err != nil {
		t.Fatalf("填写失败: %v", err)
	}

	// 使用自定义命令 "算了" 来取消
	resp, err := agent.Invoke(ctx, "算了")
	if err != nil {
		t.Fatalf("取消失败: %v", err)
	}

	// 验证取消成功
	if resp.Phase != formagent.PhaseCancelled {
		t.Errorf("期望阶段为 cancelled，实际为 %s", resp.Phase)
	}
	if !resp.Completed {
		t.Error("取消后应标记为已完成")
	}

	t.Logf("使用自定义命令 '算了' 取消成功: %s", resp.Message)
}

// TestDefaultParserCustomization 测试自定义默认解析器的关键词
func TestDefaultParserCustomization(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// 自定义默认解析器的关键词
	parser := formagent.NewDefaultCommandParser()
	parser.CancelKeywords = []string{"不干了", "放弃"}
	parser.ConfirmKeywords = []string{"就这样吧", "行"}

	agent := NewTestAgent(t, WithCommandParser(parser))

	// 填写完整信息
	if _, err := agent.Invoke(ctx, "我叫李四，邮箱 lisi@test.com，今年 30 岁"); err != nil {
		t.Fatalf("填写失败: %v", err)
	}
	if _, err := agent.Invoke(ctx, "查看一下"); err != nil {
		t.Fatalf("查看失败: %v", err)
	}

	// 使用自定义关键词 "行" 来确认
	resp, err := agent.Invoke(ctx, "行")
	if err != nil {
		t.Fatalf("确认失败: %v", err)
	}

	// 验证提交成功
	if resp.Phase != formagent.PhaseSubmitted {
		t.Errorf("期望阶段为 submitted，实际为 %s", resp.Phase)
	}
	if !resp.Completed {
		t.Error("提交后应标记为已完成")
	}

	t.Logf("使用自定义关键词 '行' 确认成功: %s", resp.Message)
}

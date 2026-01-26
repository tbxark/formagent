package testcases

import (
	"context"
	"testing"

	"github.com/TBXark/formagent"
)

// TestValidationError 测试表单验证和错误处理
func TestValidationError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent := NewTestAgent(t)

	// 用户输入基本信息
	resp, err := agent.Invoke(ctx, "我叫赵六，邮箱是 zhaoliu@testcases.com")
	if err != nil {
		t.Fatalf("第一轮对话失败: %v", err)
	}

	t.Logf("第一轮: %s", resp.Message)

	// 用户输入不合法的年龄（超出范围）
	resp, err = agent.Invoke(ctx, "我今年 150 岁")
	if err != nil {
		t.Fatalf("第二轮对话失败: %v", err)
	}

	// 验证仍在收集阶段（因为验证失败）
	if resp.Phase != formagent.PhaseCollecting {
		t.Errorf("验证失败时应保持在 collecting 阶段，实际为 %s", resp.Phase)
	}

	state := resp.FormState
	t.Logf("第二轮（年龄不合法）: %s", resp.Message)
	t.Logf("当前状态: %+v", state)

	// 用户修正年龄
	resp, err = agent.Invoke(ctx, "不好意思，我今年 35 岁")
	if err != nil {
		t.Fatalf("第三轮对话失败: %v", err)
	}

	// 验证进入确认阶段
	if resp.Phase != formagent.PhaseConfirming {
		t.Errorf("修正后应进入 confirming 阶段，实际为 %s", resp.Phase)
	}

	state = resp.FormState
	if state.Age != 35 {
		t.Errorf("期望年龄为 35，实际为 %d", state.Age)
	}

	t.Logf("第三轮（修正后）: %s", resp.Message)
	t.Logf("当前状态: %+v", state)

	// 确认提交
	resp, err = agent.Invoke(ctx, "确认")
	if err != nil {
		t.Fatalf("确认提交失败: %v", err)
	}

	if resp.Phase != formagent.PhaseSubmitted {
		t.Errorf("期望阶段为 submitted，实际为 %s", resp.Phase)
	}
	if !resp.Completed {
		t.Error("表单应该已完成")
	}

	t.Logf("最终: %s", resp.Message)
}

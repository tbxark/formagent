package testcases

import (
	"context"
	"testing"

	"github.com/tbxark/formagent/types"
)

// TestBackToEdit 测试返回编辑功能
func TestBackToEdit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent, _ := NewTestAgent(t)

	// 填写完整信息
	if _, err := agent.Invoke(ctx, "我叫孙七，邮箱 sunqi@testcases.com，今年 40 岁"); err != nil {
		t.Fatalf("填写信息失败: %v", err)
	}
	resp, err := agent.Invoke(ctx, "查看一下")
	if err != nil {
		t.Fatalf("查看失败: %v", err)
	}

	// 验证进入确认阶段
	if resp.Phase != types.PhaseConfirming {
		t.Errorf("期望阶段为 confirming，实际为 %s", resp.Phase)
	}

	t.Logf("进入确认阶段: %s", resp.Message)

	// 用户想返回修改
	resp, err = agent.Invoke(ctx, "返回修改")
	if err != nil {
		t.Fatalf("返回编辑失败: %v", err)
	}

	// 验证返回到收集阶段
	if resp.Phase != types.PhaseCollecting {
		t.Errorf("返回后应在 collecting 阶段，实际为 %s", resp.Phase)
	}

	t.Logf("返回编辑: %s", resp.Message)

	// 修改信息
	resp, err = agent.Invoke(ctx, "改一下，我今年 42 岁")
	if err != nil {
		t.Fatalf("修改信息失败: %v", err)
	}

	state := resp.FormState
	if state.Age != 42 {
		t.Errorf("期望年龄为 42，实际为 %d", state.Age)
	}

	t.Logf("修改后: %s", resp.Message)
	t.Logf("当前状态: %+v", state)
}

// TestCancel 测试取消流程
func TestCancel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent, _ := NewTestAgent(t)

	if _, err := agent.Invoke(ctx, "我叫周八"); err != nil {
		t.Fatalf("第一轮失败: %v", err)
	}
	resp, err := agent.Invoke(ctx, "取消")
	if err != nil {
		t.Fatalf("取消失败: %v", err)
	}

	// 验证取消状态
	if resp.Phase != types.PhaseCancelled {
		t.Errorf("期望阶段为 cancelled，实际为 %s", resp.Phase)
	}
	if !resp.Completed {
		t.Error("取消后应标记为已完成")
	}

	t.Logf("取消响应: %s", resp.Message)
}

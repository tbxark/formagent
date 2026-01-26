package testcases

import (
	"context"
	"testing"

	"github.com/tbxark/formagent"
)

// TestInitialState 测试使用预填充的初始状态
func TestInitialState(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent := NewTestAgent(t)

	// 从数据库或其他来源获取的初始数据
	initialState := UserRegistrationForm{
		Name: "王五",
		Age:  28,
	}

	t.Logf("初始状态: %+v", initialState)

	// 使用初始状态并让用户补充缺失信息
	resp, err := agent.InvokeWithInit(ctx, initialState, "我的邮箱是 wangwu@testcases.com")
	if err != nil {
		t.Fatalf("使用初始状态失败: %v", err)
	}

	// 验证初始状态已应用
	state := resp.FormState
	if state.Name != "王五" {
		t.Errorf("期望姓名为 '王五'，实际为 '%s'", state.Name)
	}
	if state.Age != 28 {
		t.Errorf("期望年龄为 28，实际为 %d", state.Age)
	}
	if state.Email != "wangwu@testcases.com" {
		t.Errorf("期望邮箱为 'wangwu@testcases.com'，实际为 '%s'", state.Email)
	}

	// 应该进入确认阶段（所有必填字段已填写）
	if resp.Phase != formagent.PhaseConfirming {
		t.Errorf("期望阶段为 confirming，实际为 %s", resp.Phase)
	}

	t.Logf("响应: %s", resp.Message)
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

	t.Logf("最终响应: %s", resp.Message)
}

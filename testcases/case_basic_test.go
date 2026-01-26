package testcases

import (
	"context"
	"testing"

	"github.com/TBXark/formagent"
)

// TestBasicUsage 测试基本的表单填写流程
func TestBasicUsage(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent := NewTestAgent(t)

	// 第一轮：提供部分信息
	resp, err := agent.Invoke(ctx, "我叫张三，邮箱是 zhangsan@testcases.com")
	if err != nil {
		t.Fatalf("第一轮对话失败: %v", err)
	}

	// 验证响应
	if resp.Phase != formagent.PhaseCollecting {
		t.Errorf("期望阶段为 collecting，实际为 %s", resp.Phase)
	}
	if resp.Completed {
		t.Error("表单不应该已完成")
	}

	// 验证状态
	state := resp.FormState
	if state.Name != "张三" {
		t.Errorf("期望姓名为 '张三'，实际为 '%s'", state.Name)
	}
	if state.Email != "zhangsan@testcases.com" {
		t.Errorf("期望邮箱为 'zhangsan@testcases.com'，实际为 '%s'", state.Email)
	}

	t.Logf("第一轮响应: %s", resp.Message)
	t.Logf("当前状态: %+v", state)

	// 第二轮：补充年龄
	resp, err = agent.Invoke(ctx, "我今年 25 岁")
	if err != nil {
		t.Fatalf("第二轮对话失败: %v", err)
	}

	// 验证进入确认阶段
	if resp.Phase != formagent.PhaseConfirming {
		t.Errorf("期望阶段为 confirming，实际为 %s", resp.Phase)
	}

	state = resp.FormState
	if state.Age != 25 {
		t.Errorf("期望年龄为 25，实际为 %d", state.Age)
	}

	t.Logf("第二轮响应: %s", resp.Message)
	t.Logf("当前状态: %+v", state)

	// 第三轮：确认提交
	resp, err = agent.Invoke(ctx, "确认")
	if err != nil {
		t.Fatalf("确认提交失败: %v", err)
	}

	// 验证提交成功
	if resp.Phase != formagent.PhaseSubmitted {
		t.Errorf("期望阶段为 submitted，实际为 %s", resp.Phase)
	}
	if !resp.Completed {
		t.Error("表单应该已完成")
	}

	t.Logf("最终响应: %s", resp.Message)
	t.Logf("完成状态: %v", resp.Completed)
}

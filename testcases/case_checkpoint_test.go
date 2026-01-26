package testcases

import (
	"context"
	"testing"

	"github.com/tbxark/formagent"
)

// TestCheckpointResume 测试 checkpoint 保存和恢复
func TestCheckpointResume(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent1 := NewTestAgent(t)

	// 用户开始填写表单
	resp, err := agent1.Invoke(ctx, "我叫李四，今年 30 岁")
	if err != nil {
		t.Fatalf("第一次对话失败: %v", err)
	}

	// 验证状态
	state1 := resp.FormState
	if state1.Name != "李四" {
		t.Errorf("期望姓名为 '李四'，实际为 '%s'", state1.Name)
	}
	if state1.Age != 30 {
		t.Errorf("期望年龄为 30，实际为 %d", state1.Age)
	}

	t.Logf("第一次对话: %s", resp.Message)
	t.Logf("当前状态: %+v", state1)

	// 保存 checkpoint
	checkpoint, err := agent1.CreateCheckpoint()
	if err != nil {
		t.Fatalf("创建 checkpoint 失败: %v", err)
	}
	if len(checkpoint) == 0 {
		t.Error("checkpoint 不应为空")
	}

	t.Logf("保存 checkpoint: %d bytes", len(checkpoint))

	// 创建新的 agent 实例（模拟用户稍后返回）
	agent2 := NewTestAgent(t)

	// 从 checkpoint 恢复并继续
	resp, err = agent2.InvokeWithCheckpoint(ctx, checkpoint, "我的邮箱是 lisi@testcases.com")
	if err != nil {
		t.Fatalf("从 checkpoint 恢复失败: %v", err)
	}

	// 验证状态已恢复
	state2 := resp.FormState
	if state2.Name != "李四" {
		t.Errorf("恢复后姓名应为 '李四'，实际为 '%s'", state2.Name)
	}
	if state2.Age != 30 {
		t.Errorf("恢复后年龄应为 30，实际为 %d", state2.Age)
	}
	if state2.Email != "lisi@testcases.com" {
		t.Errorf("期望邮箱为 'lisi@testcases.com'，实际为 '%s'", state2.Email)
	}

	t.Logf("恢复后对话: %s", resp.Message)
	t.Logf("恢复后状态: %+v", state2)

	// 确认提交
	resp, err = agent2.Invoke(ctx, "确认")
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

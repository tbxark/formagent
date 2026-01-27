package testcases

import (
	"context"
	"testing"

	formagent "github.com/tbxark/formagent/agent"
)

// TestMultipleCheckpoints 测试多个 checkpoint 的管理
func TestMultipleCheckpoints(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent, store := NewTestAgent(t)
	allowedPaths := (&FormSpec{}).AllowedJSONPointers()

	// 第一次中断点
	resp, err := agent.Invoke(ctx, "我叫吴九")
	if err != nil {
		t.Fatalf("步骤1失败: %v", err)
	}
	state1 := resp.FormState
	snapshot1, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取 checkpoint1 状态失败: %v", err)
	}
	checkpoint1, err := formagent.MarshalCheckpoint(snapshot1, allowedPaths)
	if err != nil {
		t.Fatalf("创建 checkpoint1 失败: %v", err)
	}

	t.Logf("步骤1: %s", resp.Message)
	t.Logf("checkpoint1: %d bytes, 状态: %+v", len(checkpoint1), state1)

	// 继续填写
	resp, err = agent.Invoke(ctx, "邮箱是 wujiu@testcases.com")
	if err != nil {
		t.Fatalf("步骤2失败: %v", err)
	}
	state2 := resp.FormState
	snapshot2, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取 checkpoint2 状态失败: %v", err)
	}
	checkpoint2, err := formagent.MarshalCheckpoint(snapshot2, allowedPaths)
	if err != nil {
		t.Fatalf("创建 checkpoint2 失败: %v", err)
	}

	t.Logf("步骤2: %s", resp.Message)
	t.Logf("checkpoint2: %d bytes, 状态: %+v", len(checkpoint2), state2)

	// 继续填写
	resp, err = agent.Invoke(ctx, "年龄 33")
	if err != nil {
		t.Fatalf("步骤3失败: %v", err)
	}
	state3 := resp.FormState
	snapshot3, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取 checkpoint3 状态失败: %v", err)
	}
	checkpoint3, err := formagent.MarshalCheckpoint(snapshot3, allowedPaths)
	if err != nil {
		t.Fatalf("创建 checkpoint3 失败: %v", err)
	}

	t.Logf("步骤3: %s", resp.Message)
	t.Logf("checkpoint3: %d bytes, 状态: %+v", len(checkpoint3), state3)

	// 验证状态演进
	if state1.Name != "吴九" || state1.Email != "" || state1.Age != 0 {
		t.Error("checkpoint1 状态不正确")
	}
	if state2.Name != "吴九" || state2.Email != "wujiu@testcases.com" || state2.Age != 0 {
		t.Error("checkpoint2 状态不正确")
	}
	if state3.Name != "吴九" || state3.Email != "wujiu@testcases.com" || state3.Age != 33 {
		t.Error("checkpoint3 状态不正确")
	}

	// 从 checkpoint2 恢复（回退到中间状态）
	t.Log("--- 从 checkpoint2 恢复 ---")
	agent2, store2 := NewTestAgent(t)
	restoredState, err := formagent.UnmarshalCheckpoint[UserRegistrationForm](checkpoint2)
	if err != nil {
		t.Fatalf("解析 checkpoint2 失败: %v", err)
	}
	if err := store2.Write(ctx, restoredState); err != nil {
		t.Fatalf("恢复 checkpoint2 失败: %v", err)
	}
	resp, err = agent2.Invoke(ctx, "查看当前状态")
	if err != nil {
		t.Fatalf("从 checkpoint2 恢复失败: %v", err)
	}

	stateRestored := resp.FormState
	if stateRestored.Name != "吴九" || stateRestored.Email != "wujiu@testcases.com" {
		t.Errorf("恢复的状态不正确: %+v", stateRestored)
	}
	if stateRestored.Age != 0 {
		t.Errorf("恢复后年龄应为 0（未填写），实际为 %d", stateRestored.Age)
	}

	t.Logf("恢复后状态: %+v", stateRestored)

	// 从恢复点继续不同的路径
	resp, err = agent2.Invoke(ctx, "年龄改成 35")
	if err != nil {
		t.Fatalf("新路径失败: %v", err)
	}
	stateFinal := resp.FormState

	if stateFinal.Age != 35 {
		t.Errorf("期望年龄为 35，实际为 %d", stateFinal.Age)
	}

	t.Logf("新路径: %s", resp.Message)
	t.Logf("最终状态: %+v", stateFinal)
}

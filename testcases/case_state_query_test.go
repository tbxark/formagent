package testcases

import (
	"context"
	"testing"

	"github.com/tbxark/formagent/types"
)

// TestStateQuery 测试状态查询功能
func TestStateQuery(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	agent, store := NewTestAgent(t)

	// 初始状态
	initialSnapshot, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取初始状态失败: %v", err)
	}
	if initialSnapshot.Phase != types.PhaseCollecting {
		t.Errorf("初始阶段应为 collecting，实际为 %s", initialSnapshot.Phase)
	}

	initialState := initialSnapshot.FormState
	if initialState.Name != "" || initialState.Email != "" || initialState.Age != 0 {
		t.Errorf("初始状态应为空，实际为 %+v", initialState)
	}

	t.Logf("初始阶段: %s", initialSnapshot.Phase)
	t.Logf("初始表单: %+v", initialState)

	// 填写部分信息
	if _, err := agent.Invoke(ctx, "我叫郑十"); err != nil {
		t.Fatalf("填写姓名失败: %v", err)
	}
	snapshot1, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取第一步状态失败: %v", err)
	}
	phase1 := snapshot1.Phase
	state1 := snapshot1.FormState

	if phase1 != types.PhaseCollecting {
		t.Errorf("阶段应为 collecting，实际为 %s", phase1)
	}
	if state1.Name != "郑十" {
		t.Errorf("期望姓名为 '郑十'，实际为 '%s'", state1.Name)
	}

	t.Logf("第一步 - 阶段: %s, 表单: %+v", phase1, state1)

	// 继续填写
	if _, err := agent.Invoke(ctx, "邮箱 zhengshi@testcases.com"); err != nil {
		t.Fatalf("填写邮箱失败: %v", err)
	}
	snapshot2, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取第二步状态失败: %v", err)
	}
	phase2 := snapshot2.Phase
	state2 := snapshot2.FormState

	if phase2 != types.PhaseCollecting {
		t.Errorf("阶段应为 collecting，实际为 %s", phase2)
	}
	if state2.Email != "zhengshi@testcases.com" {
		t.Errorf("期望邮箱为 'zhengshi@testcases.com'，实际为 '%s'", state2.Email)
	}

	t.Logf("第二步 - 阶段: %s, 表单: %+v", phase2, state2)

	// 填写完整
	if _, err := agent.Invoke(ctx, "年龄 45"); err != nil {
		t.Fatalf("填写年龄失败: %v", err)
	}
	snapshot3, err := store.Read(ctx)
	if err != nil {
		t.Fatalf("读取第三步状态失败: %v", err)
	}
	phase3 := snapshot3.Phase
	state3 := snapshot3.FormState

	if phase3 != types.PhaseConfirming {
		t.Errorf("阶段应为 confirming，实际为 %s", phase3)
	}
	if state3.Age != 45 {
		t.Errorf("期望年龄为 45，实际为 %d", state3.Age)
	}

	t.Logf("第三步 - 阶段: %s, 表单: %+v", phase3, state3)

	// 验证完整性
	if state3.Name != "郑十" || state3.Email != "zhengshi@testcases.com" || state3.Age != 45 {
		t.Errorf("最终状态不完整: %+v", state3)
	}
}

package formagent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
)

// FormAgent 是一个表单填写 Agent，实现了 Eino Runnable 接口
type FormAgent[T any] struct {
	spec              FormSpec[T]
	patchGenerator    PatchGenerator[T]
	dialogueGenerator DialogueGenerator[T]
	commandParser     CommandParser
	currentState      T
	phase             Phase
	allowedPaths      map[string]bool
}

// NewFormAgent 创建一个新的 FormAgent 实例
func NewFormAgent[T any](
	spec FormSpec[T],
	patchGen PatchGenerator[T],
	dialogGen DialogueGenerator[T],
	commandParser CommandParser,
) (*FormAgent[T], error) {
	var zero T
	allowedPaths := make(map[string]bool)
	customPaths := spec.AllowedJSONPointers()
	if len(customPaths) > 0 {
		for _, path := range customPaths {
			allowedPaths[path] = true
		}
	} else {
		allPaths := AllJSONPointerPaths[T]()
		for _, path := range allPaths {
			allowedPaths[path] = true
		}
	}
	agent := &FormAgent[T]{
		spec:              spec,
		patchGenerator:    patchGen,
		dialogueGenerator: dialogGen,
		commandParser:     commandParser,
		currentState:      zero,
		phase:             PhaseCollecting,
		allowedPaths:      allowedPaths,
	}

	return agent, nil
}

func NewToolBasedFormAgent[T any](
	spec FormSpec[T],
	chatModel model.ToolCallingChatModel,
) (*FormAgent[T], error) {
	parser, err := NewToolBasedCommandParser(chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based command parser: %w", err)
	}
	patchGen, err := NewToolBasedPatchGenerator[T](chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based patch generator: %w", err)
	}
	dialogueGen, err := NewToolBasedDialogueGenerator[T](chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based dialogue generator: %w", err)
	}
	return NewFormAgent[T](
		spec,
		patchGen,
		dialogueGen,
		parser,
	)
}

// Invoke 实现 Eino Runnable 接口
func (a *FormAgent[T]) Invoke(ctx context.Context, input string) (*Response[T], error) {
	ctx = callbacks.EnsureRunInfo(ctx, "FormAgent", "Agent")
	ctx = callbacks.OnStart(ctx, map[string]any{
		"input":      input,
		"phase":      string(a.phase),
		"form_state": a.currentState,
	})

	defer func() {
		if r := recover(); r != nil {
			callbacks.OnError(ctx, fmt.Errorf("panic in FormAgent.Invoke: %v", r))
			panic(r)
		}
	}()

	response, err := a.runInternal(ctx, input)
	if err != nil {
		callbacks.OnError(ctx, err)
		return nil, err
	}

	callbacks.OnEnd(ctx, map[string]any{
		"response":  response,
		"phase":     string(response.Phase),
		"completed": response.Completed,
	})

	return response, nil
}

func (a *FormAgent[T]) runInternal(ctx context.Context, input string) (*Response[T], error) {
	cmd, err := a.commandParser.ParseCommand(ctx, input)
	if err != nil {
		return a.handleError(ctx, fmt.Errorf("failed to parse command: %w", err), input, false)
	}
	if cmd != CommandNone {
		return a.handleCommand(ctx, cmd, input)
	}

	missingFields := a.spec.MissingFacts(a.currentState)
	fieldGuidance := make(map[string]string)
	for _, field := range missingFields {
		guidance := a.spec.FieldGuide(field.JSONPointer)
		if guidance != "" {
			fieldGuidance[field.JSONPointer] = guidance
		}
	}

	patchReq := PatchRequest[T]{
		UserInput:     input,
		CurrentState:  a.currentState,
		AllowedPaths:  a.getAllowedPathsList(),
		MissingFields: missingFields,
		FieldGuidance: fieldGuidance,
	}

	updateArgs, err := a.patchGenerator.GeneratePatch(ctx, patchReq)
	if err != nil {
		return a.handleError(ctx, fmt.Errorf("failed to generate patch: %w", err), input, false)
	}

	patchApplied := false
	if len(updateArgs.Ops) > 0 {
		newState, err := ApplyRFC6902(a.currentState, updateArgs.Ops, a.allowedPaths)
		if err != nil {
			return a.handleError(ctx, fmt.Errorf("failed to apply patch: %w", err), input, false)
		}
		a.currentState = newState
		patchApplied = true
	}

	missingFields = a.spec.MissingFacts(a.currentState)
	validationErrors := a.spec.ValidateFacts(a.currentState)

	if a.phase == PhaseCollecting && len(missingFields) == 0 && len(validationErrors) == 0 {
		a.phase = PhaseConfirming
	}

	dialogueReq := DialogueRequest[T]{
		CurrentState:     a.currentState,
		Phase:            a.phase,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
		LastUserInput:    input,
		PatchApplied:     patchApplied,
	}

	plan, err := a.dialogueGenerator.GenerateDialogue(ctx, dialogueReq)
	if err != nil {
		return a.handleError(ctx, fmt.Errorf("failed to generate dialogue: %w", err), input, patchApplied)
	}

	return &Response[T]{
		Message:   plan.Message,
		Phase:     a.phase,
		FormState: a.currentState,
		Completed: a.phase == PhaseSubmitted || a.phase == PhaseCancelled,
		Metadata: map[string]string{
			"suggested_action": plan.SuggestedAction,
		},
	}, nil
}

func (a *FormAgent[T]) handleCommand(ctx context.Context, cmd Command, input string) (*Response[T], error) {
	var message string
	var completed bool

	switch cmd {
	case CommandCancel:
		a.phase = PhaseCancelled
		message = "表单填写已取消。"
		completed = true

	case CommandConfirm:
		if a.phase != PhaseConfirming {
			message = "请先完成所有必填字段后再确认提交。"
		} else {
			validationErrors := a.spec.ValidateFacts(a.currentState)
			if len(validationErrors) > 0 {
				message = "表单验证失败，请修正错误后再提交。"
			} else {
				if err := a.spec.Submit(ctx, a.currentState); err != nil {
					return nil, fmt.Errorf("failed to submit form: %w", err)
				}
				a.phase = PhaseSubmitted
				message = "表单已成功提交！"
				completed = true
			}
		}

	case CommandBack:
		if a.phase == PhaseConfirming {
			a.phase = PhaseCollecting
			message = "已返回编辑模式，您可以继续修改表单内容。"
		} else {
			message = "当前不在确认阶段，无需返回。"
		}

	default:
		return nil, fmt.Errorf("unknown command: %s", cmd)
	}

	return &Response[T]{
		Message:   message,
		Phase:     a.phase,
		FormState: a.currentState,
		Completed: completed,
		Metadata:  map[string]string{},
	}, nil
}

func (a *FormAgent[T]) handleError(ctx context.Context, err error, lastInput string, patchApplied bool) (*Response[T], error) {
	message := fmt.Sprintf("抱歉，处理您的输入时遇到了问题：%s", err.Error())

	missingFields := a.spec.MissingFacts(a.currentState)
	validationErrors := a.spec.ValidateFacts(a.currentState)

	dialogueReq := DialogueRequest[T]{
		CurrentState:     a.currentState,
		Phase:            a.phase,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
		LastUserInput:    lastInput,
		PatchApplied:     patchApplied,
	}

	plan, dialogueErr := a.dialogueGenerator.GenerateDialogue(ctx, dialogueReq)
	if dialogueErr == nil && plan != nil {
		message = plan.Message
	}

	return &Response[T]{
		Message:   message,
		Phase:     a.phase,
		FormState: a.currentState,
		Completed: false,
		Metadata: map[string]string{
			"error": err.Error(),
		},
	}, nil
}

func (a *FormAgent[T]) getAllowedPathsList() []string {
	paths := make([]string, 0, len(a.allowedPaths))
	for path := range a.allowedPaths {
		paths = append(paths, path)
	}
	return paths
}

func (a *FormAgent[T]) CreateCheckpoint() ([]byte, error) {
	checkpoint := Checkpoint[T]{
		Version:      "1.0",
		Phase:        a.phase,
		FormState:    a.currentState,
		Timestamp:    time.Now(),
		AllowedPaths: a.getAllowedPathsList(),
	}

	data, err := json.Marshal(checkpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	return data, nil
}

// RestoreCheckpoint 从 checkpoint 恢复状态
func (a *FormAgent[T]) RestoreCheckpoint(checkpointData []byte) error {
	var checkpoint Checkpoint[T]
	if err := json.Unmarshal(checkpointData, &checkpoint); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	if checkpoint.Version != "1.0" {
		return fmt.Errorf("incompatible checkpoint version: %s (expected 1.0)", checkpoint.Version)
	}

	a.phase = checkpoint.Phase
	a.currentState = checkpoint.FormState

	a.allowedPaths = make(map[string]bool)
	for _, path := range checkpoint.AllowedPaths {
		a.allowedPaths[path] = true
	}

	return nil
}

// InvokeWithCheckpoint 从 checkpoint 恢复后执行
func (a *FormAgent[T]) InvokeWithCheckpoint(ctx context.Context, checkpointData []byte, input string) (*Response[T], error) {
	if err := a.RestoreCheckpoint(checkpointData); err != nil {
		return nil, err
	}
	return a.Invoke(ctx, input)
}

// SetInitialState 设置初始状态
func (a *FormAgent[T]) SetInitialState(initial T) error {
	patches, err := generatePatchesFromInitial(a.currentState, initial)
	if err != nil {
		return fmt.Errorf("failed to generate patches from initial values: %w", err)
	}

	if len(patches) > 0 {
		newState, err := ApplyRFC6902(a.currentState, patches, a.allowedPaths)
		if err != nil {
			return fmt.Errorf("failed to apply initial values: %w", err)
		}
		a.currentState = newState
	}

	return nil
}

// InvokeWithInit 设置初始状态后执行
func (a *FormAgent[T]) InvokeWithInit(ctx context.Context, initial T, input string) (*Response[T], error) {
	if err := a.SetInitialState(initial); err != nil {
		return nil, err
	}
	return a.Invoke(ctx, input)
}

// GetCurrentState 获取当前表单状态
func (a *FormAgent[T]) GetCurrentState() T {
	return a.currentState
}

// GetPhase 获取当前阶段
func (a *FormAgent[T]) GetPhase() Phase {
	return a.phase
}

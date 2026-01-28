package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/tbxark/formagent/command"
	"github.com/tbxark/formagent/dialogue"
	"github.com/tbxark/formagent/patch"
	"github.com/tbxark/formagent/types"
)

type FormFlow[T any] struct {
	schema            string
	spec              FormSpec[T]
	patchGenerator    patch.Generator[T]
	dialogueGenerator dialogue.Generator[T]
	commandParser     command.Parser
	stateStore        StateReadWriter[T]
}

func NewFormFlow[T any](
	spec FormSpec[T],
	patchGen patch.Generator[T],
	dialogGen dialogue.Generator[T],
	commandParser command.Parser,
	stateStore StateReadWriter[T],
) (*FormFlow[T], error) {
	schema, err := spec.JsonSchema()
	if err != nil {
		return nil, err
	}
	agent := &FormFlow[T]{
		schema:            schema,
		spec:              spec,
		patchGenerator:    patchGen,
		dialogueGenerator: dialogGen,
		commandParser:     commandParser,
		stateStore:        stateStore,
	}

	return agent, nil
}

func NewToolBasedFormFlow[T any](
	spec FormSpec[T],
	chatModel model.ToolCallingChatModel,
	stateStore StateReadWriter[T],
) (*FormFlow[T], error) {
	parser, err := command.NewToolBasedCommandParser(chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based command parser: %w", err)
	}
	patchGen, err := patch.NewToolBasedPatchGenerator[T](chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based patch generator: %w", err)
	}
	dialogueGen, err := dialogue.NewToolBasedDialogueGenerator[T](chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based dialogue generator: %w", err)
	}
	return NewFormFlow[T](
		spec,
		patchGen,
		dialogueGen,
		parser,
		stateStore,
	)
}

func (a *FormFlow[T]) Invoke(ctx context.Context, input *Request) (*Response[T], error) {
	state, err := a.stateStore.Read(ctx)
	if err != nil {
		return nil, err
	}
	if state.Phase == "" {
		state.Phase = types.PhaseCollecting
	}

	response, nextState, err := a.runInternal(ctx, input.UserInput, state)
	if err != nil {
		return nil, err
	}
	err = a.stateStore.Write(ctx, nextState)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *FormFlow[T]) runInternal(ctx context.Context, input string, state *State[T]) (*Response[T], *State[T], error) {
	cmd, err := a.commandParser.ParseCommand(ctx, input)
	if err != nil {
		return a.handleError(ctx, fmt.Errorf("failed to parse command: %w", err), state)
	}
	if cmd == command.Confirm || cmd == command.Cancel {
		return a.handleCommand(ctx, cmd, state)
	}

	missingFields := a.spec.MissingFacts(state.FormState)

	patchReq := patch.Request[T]{
		AssistantQuestion: state.LatestQuestion,
		UserAnswer:        input,
		CurrentState:      state.FormState,
		StateSchema:       a.schema,
	}

	updateArgs, err := a.patchGenerator.GeneratePatch(ctx, &patchReq)
	if err != nil {
		return a.handleError(ctx, fmt.Errorf("failed to generate patch: %w", err), state)
	}

	if len(updateArgs.Ops) > 0 {
		newState, pErr := patch.ApplyRFC6902(state.FormState, updateArgs.Ops)
		if pErr != nil {
			return a.handleError(ctx, fmt.Errorf("failed to apply patch: %w", pErr), state)
		}
		state.FormState = newState
	}

	missingFields = a.spec.MissingFacts(state.FormState)
	validationErrors := a.spec.ValidateFacts(state.FormState)

	if state.Phase == types.PhaseCollecting && len(missingFields) == 0 && len(validationErrors) == 0 {
		state.Phase = types.PhaseConfirming
	}

	dialogueReq := dialogue.Request[T]{
		CurrentState:     state.FormState,
		Phase:            state.Phase,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
	}

	plan, err := a.dialogueGenerator.GenerateDialogue(ctx, &dialogueReq)
	if err != nil {
		return a.handleError(ctx, fmt.Errorf("failed to generate dialogue: %w", err), state)
	}
	state.LatestQuestion = plan.Message

	return &Response[T]{
		Message:   plan.Message,
		Phase:     state.Phase,
		FormState: state.FormState,
		Completed: state.Phase == types.PhaseSubmitted || state.Phase == types.PhaseCancelled,
	}, state, nil
}

func (a *FormFlow[T]) handleCommand(ctx context.Context, cmd command.Command, state *State[T]) (*Response[T], *State[T], error) {
	var message string
	var completed bool

	switch cmd {
	case command.Cancel:
		state.Phase = types.PhaseCancelled
		message = "表单填写已取消。"
		completed = true

	case command.Confirm:
		if state.Phase != types.PhaseConfirming {
			message = "请先完成所有必填字段后再确认提交。"
		} else {
			validationErrors := a.spec.ValidateFacts(state.FormState)
			if len(validationErrors) > 0 {
				message = "表单验证失败，请修正错误后再提交。"
			} else {
				if err := a.spec.Submit(ctx, state.FormState); err != nil {
					return nil, state, fmt.Errorf("failed to submit form: %w", err)
				}
				state.Phase = types.PhaseSubmitted
				message = "表单已成功提交！"
				completed = true
			}
		}

	case command.Back:
		if state.Phase == types.PhaseConfirming {
			state.Phase = types.PhaseCollecting
			message = "已返回编辑模式，您可以继续修改表单内容。"
		} else {
			message = "当前不在确认阶段，无需返回。"
		}

	default:
		return nil, state, fmt.Errorf("unknown command: %s", cmd)
	}

	return &Response[T]{
		Message:   message,
		Phase:     state.Phase,
		FormState: state.FormState,
		Completed: completed,
		Metadata:  map[string]string{},
	}, state, nil
}

func (a *FormFlow[T]) handleError(ctx context.Context, err error, state *State[T]) (*Response[T], *State[T], error) {
	message := fmt.Sprintf("抱歉，处理您的输入时遇到了问题：%s", err.Error())

	missingFields := a.spec.MissingFacts(state.FormState)
	validationErrors := a.spec.ValidateFacts(state.FormState)

	dialogueReq := dialogue.Request[T]{
		CurrentState:     state.FormState,
		Phase:            state.Phase,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
	}

	plan, dialogueErr := a.dialogueGenerator.GenerateDialogue(ctx, &dialogueReq)
	if dialogueErr == nil && plan != nil {
		message = plan.Message
	}

	return &Response[T]{
		Message:   message,
		Phase:     state.Phase,
		FormState: state.FormState,
		Completed: false,
		Metadata: map[string]string{
			"error": err.Error(),
		},
	}, state, nil
}

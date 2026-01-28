package agent

import (
	"context"
	"fmt"
	"log/slog"

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
}

func NewFormFlow[T any](
	spec FormSpec[T],
	manager FormManager[T],
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
	}

	return agent, nil
}

func NewToolBasedFormFlow[T any](
	spec FormSpec[T],
	manager FormManager[T],
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
	dialogueGen := dialogue.NewToolBasedDialogueGenerator[T](chatModel)
	return NewFormFlow[T](
		spec,
		manager,
		patchGen,
		dialogueGen,
		parser,
		stateStore,
	)
}

func (a *FormFlow[T]) Invoke(ctx context.Context, input *Request[T]) (*Response[T], error) {
	if input.State.Phase == "" {
		input.State.Phase = types.PhaseCollecting
	}
	response, err := a.runInternal(ctx, input)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *FormFlow[T]) runInternal(ctx context.Context, input *Request[T]) (*Response[T], error) {

	// command
	slog.Debug("Loaded state", "state", input.State.FormState)
	cmd, err := a.commandParser.ParseCommand(ctx, input.UserInput)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to parse command: %w", err), input)
	}
	slog.Debug("Parsed command", "command", cmd, "input", input)
	if cmd == command.Confirm || cmd == command.Cancel {
		return a.handleCommand(cmd, input)
	}

	// patch

	patchReq := patch.Request[T]{
		AssistantQuestion: input.State.LatestQuestion,
		UserAnswer:        input.UserInput,
		CurrentState:      input.State.FormState,
		StateSchema:       a.schema,
	}
	updateArgs, err := a.patchGenerator.GeneratePatch(ctx, &patchReq)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to generate patch: %w", err), input)
	}
	slog.Debug("Applying patch", "ops", updateArgs.Ops, "to_state", input.State.FormState)
	newState, err := patch.ApplyRFC6902(input.State.FormState, updateArgs.Ops)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to apply patch: %w", err), input)
	}
	input.State.FormState = newState
	slog.Debug("Applied patch", "to_state", input.State.FormState)

	// dialogue

	missingFields := a.spec.MissingFacts(input.State.FormState)
	validationErrors := a.spec.ValidateFacts(input.State.FormState)
	if input.State.Phase == types.PhaseCollecting && len(missingFields) == 0 && len(validationErrors) == 0 {
		input.State.Phase = types.PhaseConfirming
	}
	dialogueReq := dialogue.Request[T]{
		CurrentState:     input.State.FormState,
		Phase:            input.State.Phase,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
	}

	slog.Debug("Generating dialogue", "request", dialogueReq)
	question, err := a.dialogueGenerator.GenerateDialogue(ctx, &dialogueReq)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to generate dialogue: %w", err), input)
	}
	input.State.LatestQuestion = question
	slog.Debug("Generated dialogue", "question", question)

	return &Response[T]{
		Message: question,
		State:   input.State,
	}, nil
}

func (a *FormFlow[T]) handleCommand(cmd command.Command, input *Request[T]) (*Response[T], error) {
	resp := &Response[T]{
		Message:  "",
		State:    input.State,
		Metadata: map[string]string{},
	}
	switch cmd {
	case command.Cancel:
		resp.Message = "表单填写已取消。"
		resp.State = &State[T]{
			Phase: types.PhaseCancelled,
		}
	case command.Confirm:
		if resp.State.Phase != types.PhaseConfirming {
			resp.Message = "请先完成所有必填字段后再确认提交。"
		} else {
			validationErrors := a.spec.ValidateFacts(resp.State.FormState)
			if len(validationErrors) > 0 {
				resp.Message = "表单验证失败，请修正错误后再提交。"
			} else {
				resp.Message = "表单已成功提交，谢谢！"
				resp.State.Phase = types.PhaseConfirmed
			}
		}
	default:
		return resp, nil
	}
	return resp, nil
}

func (a *FormFlow[T]) handleError(err error, input *Request[T]) (*Response[T], error) {
	message := fmt.Sprintf("抱歉，处理您的输入时遇到了问题：%s", err.Error())

	return &Response[T]{
		Message: message,
		State:   input.State,
		Metadata: map[string]string{
			"error": err.Error(),
		},
	}, nil
}

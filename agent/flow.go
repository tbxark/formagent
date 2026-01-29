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
	commandParser     command.Parser[T]
}

func NewFormFlow[T any](spec FormSpec[T], patchGen patch.Generator[T], dialogGen dialogue.Generator[T], commandParser command.Parser[T]) (*FormFlow[T], error) {
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
	chatModel model.ToolCallingChatModel,
) (*FormFlow[T], error) {
	parser, err := command.NewToolBasedCommandParser[T](chatModel)
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
		patchGen,
		dialogueGen,
		parser,
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

	missingFields := a.spec.MissingFacts(input.State.FormState)
	validationErrors := a.spec.ValidateFacts(input.State.FormState)
	toolRequest := &types.ToolRequest[T]{
		State:            input.State.FormState,
		Phase:            input.State.Phase,
		Messages:         input.ChatHistory,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
	}

	// command
	slog.Debug("Parsing command", "request", toolRequest.State)
	cmd, err := a.commandParser.ParseCommand(ctx, toolRequest)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to parse command: %w", err), input)
	}
	slog.Debug("Parsed command", "command", cmd, "input", input)
	switch cmd {
	case command.Confirm:
		if len(missingFields) == 0 && len(validationErrors) == 0 {
			return a.handleCommand(cmd, input)
		}
	case command.Cancel:
		return a.handleCommand(cmd, input)
	case command.Edit:
		// patch
		toolRequest.StateSchema = a.schema
		slog.Debug("Requesting patch generation")
		updateArgs, pErr := a.patchGenerator.GeneratePatch(ctx, toolRequest)
		if pErr != nil {
			return a.handleError(fmt.Errorf("failed to generate patch: %w", pErr), input)
		}
		slog.Debug("Applying patch", "ops", updateArgs.Ops)
		newState, pErr := patch.ApplyRFC6902(input.State.FormState, updateArgs.Ops)
		if pErr != nil {
			return a.handleError(fmt.Errorf("failed to apply patch: %w", pErr), input)
		}
		input.State.FormState = newState
		slog.Debug("Applied patch", "phase", input.State.Phase, "to_state", input.State.FormState)
	case command.DoNothing:
		break
	}

	// dialogue
	toolRequest.MissingFields = a.spec.MissingFacts(input.State.FormState)
	toolRequest.ValidationErrors = a.spec.ValidateFacts(input.State.FormState)
	slog.Debug("Generating dialogue")
	question, err := a.dialogueGenerator.GenerateDialogue(ctx, toolRequest)
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
		resp.State.Phase = types.PhaseCancelled
	case command.Confirm:
		resp.Message = "表单已成功提交，谢谢！"
		resp.State.Phase = types.PhaseConfirmed
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

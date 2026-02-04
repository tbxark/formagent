package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/tbxark/formagent/dialogue"
	"github.com/tbxark/formagent/indent"
	"github.com/tbxark/formagent/patch"
	"github.com/tbxark/formagent/types"
)

type FormFlow[T any] struct {
	Schema            string
	PatchHook         func(T, []patch.Operation) ([]patch.Operation, error)
	Spec              FormSpec[T]
	PatchGenerator    patch.Generator[T]
	DialogueGenerator dialogue.Generator[T]
	IndentRecognizer  indent.Recognizer[T]
}

func NewFormFlow[T any](spec FormSpec[T], patchGen patch.Generator[T], dialogGen dialogue.Generator[T], indentRecognizer indent.Recognizer[T]) (*FormFlow[T], error) {
	schema, err := spec.JsonSchema()
	if err != nil {
		return nil, err
	}
	agent := &FormFlow[T]{
		Schema:            schema,
		Spec:              spec,
		PatchGenerator:    patchGen,
		DialogueGenerator: dialogGen,
		IndentRecognizer:  indentRecognizer,
	}

	return agent, nil
}

func NewToolBasedFormFlow[T any](
	spec FormSpec[T],
	chatModel model.ToolCallingChatModel,
) (*FormFlow[T], error) {
	parser, err := indent.NewToolBasedIntentRecognizer[T](chatModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool-based indent parser: %w", err)
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

	missingFields := a.Spec.MissingFacts(ctx, input.State.FormState)
	validationErrors := a.Spec.ValidateFacts(ctx, input.State.FormState)
	toolRequest := &types.ToolRequest[T]{
		State:            input.State.FormState,
		StateSummary:     a.Spec.Summary(ctx, input.State.FormState),
		Phase:            input.State.Phase,
		Messages:         input.ChatHistory,
		MissingFields:    missingFields,
		ValidationErrors: validationErrors,
	}

	// indent
	slog.Debug("Parsing indent", "request", toolRequest.State)
	cmd, err := a.IndentRecognizer.RecognizerIntent(ctx, toolRequest)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to parse indent: %w", err), input)
	}
	slog.Debug("Parsed indent", "indent", cmd, "input", input)
	switch cmd {
	case indent.Confirm:
		if len(missingFields) == 0 && len(validationErrors) == 0 {
			return a.handleCommand(cmd, input)
		}
	case indent.Cancel:
		return a.handleCommand(cmd, input)
	case indent.Edit:
		// patch
		toolRequest.StateSchema = a.Schema
		slog.Debug("Requesting patch generation")
		updateArgs, pErr := a.PatchGenerator.GeneratePatch(ctx, toolRequest)
		if pErr != nil {
			return a.handleError(fmt.Errorf("failed to generate patch: %w", pErr), input)
		}
		if a.PatchHook != nil {
			updateArgs.Ops, pErr = a.PatchHook(input.State.FormState, updateArgs.Ops)
			if pErr != nil {
				return a.handleError(fmt.Errorf("failed to apply patch hook: %w", pErr), input)
			}
		}
		slog.Debug("Applying patch", "ops", updateArgs.Ops)
		newState, pErr := patch.ApplyRFC6902(input.State.FormState, updateArgs.Ops)
		if pErr != nil {
			return a.handleError(fmt.Errorf("failed to apply patch: %w", pErr), input)
		}
		input.State.FormState = newState
		slog.Debug("Applied patch", "phase", input.State.Phase, "to_state", input.State.FormState)
	case indent.DoNothing:
		break
	}

	// dialogue
	toolRequest.MissingFields = a.Spec.MissingFacts(ctx, input.State.FormState)
	toolRequest.ValidationErrors = a.Spec.ValidateFacts(ctx, input.State.FormState)
	slog.Debug("Generating dialogue")
	question, err := a.DialogueGenerator.GenerateDialogue(ctx, toolRequest)
	if err != nil {
		return a.handleError(fmt.Errorf("failed to generate dialogue: %w", err), input)
	}
	slog.Debug("Generated dialogue", "question", question)

	return &Response[T]{
		Message: question,
		State:   input.State,
	}, nil
}

func (a *FormFlow[T]) handleCommand(cmd indent.Intent, input *Request[T]) (*Response[T], error) {
	resp := &Response[T]{
		Message:  "",
		State:    input.State,
		Metadata: map[string]string{},
	}
	switch cmd {
	case indent.Cancel:
		resp.Message = "表单填写已取消。"
		resp.State.Phase = types.PhaseCancelled
	case indent.Confirm:
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

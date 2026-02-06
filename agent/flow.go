package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/dialogue"
	"github.com/tbxark/formagent/indent"
	"github.com/tbxark/formagent/patch"
	"github.com/tbxark/formagent/types"
)

type FormFlow[T any] struct {
	PatchHook         func(T, []patch.Operation) ([]patch.Operation, error)
	Spec              FormSpec[T]
	PatchGenerator    patch.Generator[T]
	DialogueGenerator dialogue.Generator[T]
	IndentRecognizer  indent.Recognizer[T]
}

func NewFormFlow[T any](spec FormSpec[T], patchGen patch.Generator[T], dialogGen dialogue.Generator[T], indentRecognizer indent.Recognizer[T]) *FormFlow[T] {
	return &FormFlow[T]{
		Spec:              spec,
		PatchGenerator:    patchGen,
		DialogueGenerator: dialogGen,
		IndentRecognizer:  indentRecognizer,
	}
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
	), nil
}

func (a *FormFlow[T]) Invoke(ctx context.Context, input *Request[T]) (*Response[T], error) {
	toolRequest := a.newToolRequest(ctx, input)
	response, err := a.runInternal(ctx, toolRequest)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *FormFlow[T]) Stream(ctx context.Context, input *Request[T]) (*StreamResponse[T], error) {
	toolRequest := a.newToolRequest(ctx, input)
	response, err := a.runInternalStream(ctx, toolRequest)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (a *FormFlow[T]) newToolRequest(ctx context.Context, input *Request[T]) *types.ToolRequest[T] {
	if input.State.Phase == "" {
		input.State.Phase = types.PhaseCollecting
	}
	return &types.ToolRequest[T]{
		State:            input.State.FormState,
		StateSummary:     a.Spec.Summary(ctx, input.State.FormState),
		Phase:            input.State.Phase,
		Messages:         input.ChatHistory,
		MissingFields:    a.Spec.MissingFacts(ctx, input.State.FormState),
		ValidationErrors: a.Spec.ValidateFacts(ctx, input.State.FormState),
		Extra:            make(map[string]any),
	}
}

func (a *FormFlow[T]) runInternal(ctx context.Context, request *types.ToolRequest[T]) (*Response[T], error) {
	commandResp, err := a.preprocessRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	if commandResp != nil {
		return commandResp, nil
	}

	// dialogue
	slog.Debug("Generating dialogue")
	question, err := a.DialogueGenerator.GenerateDialogue(ctx, request)
	if err != nil {
		return nil, err
	}
	slog.Debug("Generated dialogue", "question", question)

	return &Response[T]{
		Message: question,
		State: &State[T]{
			Phase:     request.Phase,
			FormState: request.State,
		},
	}, nil
}

func (a *FormFlow[T]) runInternalStream(ctx context.Context, request *types.ToolRequest[T]) (*StreamResponse[T], error) {
	commandResp, err := a.preprocessRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	if commandResp != nil {
		return &StreamResponse[T]{
			MessageStream: schema.StreamReaderFromArray([]string{commandResp.Message}),
			State:         commandResp.State,
			Metadata:      commandResp.Metadata,
		}, nil
	}

	slog.Debug("Generating dialogue stream")
	stream, err := a.DialogueGenerator.GenerateDialogueStream(ctx, request)
	if err != nil {
		return nil, err
	}

	return &StreamResponse[T]{
		MessageStream: stream,
		State: &State[T]{
			Phase:     request.Phase,
			FormState: request.State,
		},
	}, nil
}

func (a *FormFlow[T]) preprocessRequest(ctx context.Context, request *types.ToolRequest[T]) (*Response[T], error) {
	// indent
	slog.Debug("Parsing indent", "request", request.State)
	cmd, err := a.IndentRecognizer.RecognizerIntent(ctx, request)
	if err != nil {
		return nil, err
	}
	slog.Debug("Parsed indent", "indent", cmd)

	switch cmd {
	case indent.Confirm:
		if len(request.MissingFields) == 0 && len(request.ValidationErrors) == 0 {
			return a.handleCommand(cmd, request)
		}
	case indent.Cancel:
		return a.handleCommand(cmd, request)
	case indent.Edit:
		// patch
		slog.Debug("Requesting patch generation")
		updateArgs, pErr := a.PatchGenerator.GeneratePatch(ctx, request)
		if pErr != nil {
			return nil, pErr
		}
		if a.PatchHook != nil {
			updateArgs.Ops, pErr = a.PatchHook(request.State, updateArgs.Ops)
			if pErr != nil {
				return nil, pErr
			}
		}
		if len(updateArgs.Ops) == 0 {
			break
		}
		slog.Debug("Applying patch", "ops", updateArgs.Ops)
		newState, pErr := patch.ApplyRFC6902(request.State, updateArgs.Ops)
		if pErr != nil {
			return nil, pErr
		}
		// update state
		request.State = newState
		request.StateSummary = a.Spec.Summary(ctx, request.State)
		request.MissingFields = a.Spec.MissingFacts(ctx, request.State)
		request.ValidationErrors = a.Spec.ValidateFacts(ctx, request.State)
		request.Extra["ops"] = updateArgs.Ops
	case indent.DoNothing:
		break
	}

	return nil, nil
}

func (a *FormFlow[T]) handleCommand(cmd indent.Intent, request *types.ToolRequest[T]) (*Response[T], error) {
	resp := &Response[T]{
		Message: "",
		State: &State[T]{
			Phase:     request.Phase,
			FormState: request.State,
		},
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

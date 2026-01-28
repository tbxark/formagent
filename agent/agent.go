package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

var _ adk.Agent = (*Agent[any])(nil)

type Agent[T any] struct {
	name        string
	description string
	flow        *FormFlow[T]
	store       StateReadWriter[T]
	manager     FormManager[T]
}

func NewAgent[T any](name, description string, flow *FormFlow[T], store StateReadWriter[T], manager FormManager[T]) *Agent[T] {
	return &Agent[T]{
		name:        name,
		description: description,
		flow:        flow,
		store:       store,
		manager:     manager,
	}
}

func (a *Agent[T]) Name(ctx context.Context) string {
	return a.name
}

func (a *Agent[T]) Description(ctx context.Context) string {
	return a.description
}

func (a *Agent[T]) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer func() {
			if e := recover(); e != nil {
				gen.Send(&adk.AgentEvent{
					Err: fmt.Errorf("recover from panic: %v", e),
				})
			}
			gen.Close()
		}()
		if len(input.Messages) == 0 {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("no messages in input"),
			})
			return
		}
		state, err := a.store.Read(ctx)
		if err != nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("failed to read state: %w", err),
			})
			return
		}
		resp, err := a.flow.Invoke(ctx, &Request[T]{
			State:       state,
			ChatHistory: input.Messages,
		})
		if err != nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("flow invoke failed: %w", err),
			})
			return
		}
		switch resp.State.Phase {
		case types.PhaseConfirmed:
			err = errors.Join(
				a.manager.Submit(ctx, resp.State.FormState),
				a.store.Remove(ctx),
			)
		case types.PhaseCollecting:
			err = a.store.Write(ctx, resp.State)
		case types.PhaseCancelled:
			err = errors.Join(
				a.manager.Cancel(ctx, resp.State.FormState),
				a.store.Remove(ctx),
			)
		}
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message: &schema.Message{
						Role:    schema.Assistant,
						Content: resp.Message,
					},
					Role: schema.Assistant,
				},
			},
		})
	}()
	return iter
}

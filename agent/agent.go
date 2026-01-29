package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

var _ adk.Agent = (*Agent[any])(nil)

type Agent[T any] struct {
	name        string
	description string
	flow        *FormFlow[T]
	store       StateReadWriter[T]
}

type Option[T any] func(a *Agent[T])

func NewAgent[T any](name, description string, flow *FormFlow[T], store StateReadWriter[T], options ...Option[T]) *Agent[T] {
	ag := &Agent[T]{
		name:        name,
		description: description,
		flow:        flow,
		store:       store,
	}
	for _, opt := range options {
		if opt != nil {
			opt(ag)
		}
	}
	return ag
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
		state, err := a.store.Load(ctx)
		if err != nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("failed to load session: %w", err),
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
		err = a.store.Save(ctx, resp.State)
		if err != nil {
			gen.Send(&adk.AgentEvent{Err: err})
			return
		}
		gen.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message: &schema.Message{
						Role:    schema.Assistant,
						Content: resp.Message,
						Extra: map[string]any{
							"phase": string(resp.State.Phase),
						},
					},
					Role: schema.Assistant,
				},
			},
		})
	}()
	return iter
}

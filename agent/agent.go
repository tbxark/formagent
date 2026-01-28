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
}

func NewAgent[T any](name, description string, flow *FormFlow[T]) *Agent[T] {
	return &Agent[T]{
		name:        name,
		description: description,
		flow:        flow,
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
			e := recover()
			if e != nil {
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
		resp, err := a.flow.Invoke(ctx, &Request{
			UserInput: input.Messages[len(input.Messages)-1].Content,
		})
		if err != nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("flow invoke failed: %w", err),
			})
			return
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

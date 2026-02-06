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

func NewAgent[T any](name, description string, flow *FormFlow[T], store StateReadWriter[T]) *Agent[T] {
	return &Agent[T]{
		name:        name,
		description: description,
		flow:        flow,
		store:       store,
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
		state, err := a.store.Load(ctx)
		if err != nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("failed to load session: %w", err),
			})
			return
		}
		req := &Request[T]{
			State:       state,
			ChatHistory: input.Messages,
		}
		if input.EnableStreaming {
			streamResp, invokeErr := a.flow.Stream(ctx, req)
			if invokeErr != nil {
				gen.Send(&adk.AgentEvent{Err: fmt.Errorf("flow stream invoke failed: %w", invokeErr)})
				return
			}
			if saveErr := a.store.Save(ctx, streamResp.State); saveErr != nil {
				gen.Send(&adk.AgentEvent{Err: saveErr})
				return
			}
			msgStream := schema.StreamReaderWithConvert[string, *schema.Message](streamResp.MessageStream, func(content string) (*schema.Message, error) {
				return &schema.Message{
					Role:    schema.Assistant,
					Content: content,
					Extra: map[string]any{
						"phase": string(streamResp.State.Phase),
					},
				}, nil
			})
			gen.Send(&adk.AgentEvent{
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						IsStreaming:   true,
						MessageStream: msgStream,
						Role:          schema.Assistant,
					},
				},
			})
			return
		}

		resp, invokeErr := a.flow.Invoke(ctx, req)
		if invokeErr != nil {
			gen.Send(&adk.AgentEvent{
				Err: fmt.Errorf("flow invoke failed: %w", invokeErr),
			})
			return
		}
		if saveErr := a.store.Save(ctx, resp.State); saveErr != nil {
			gen.Send(&adk.AgentEvent{Err: saveErr})
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

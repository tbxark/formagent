package agent

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type StateReadWriter[T any] interface {
	Load(ctx context.Context) (*State[T], error)
	Save(ctx context.Context, state *State[T]) error
	Clear(ctx context.Context) error
}

type stateKeyContext struct{}

func WithStateKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, stateKeyContext{}, key)
}

func StateKeyFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(stateKeyContext{})
	if value == nil {
		return "", false
	}
	key, ok := value.(string)
	return key, ok
}

type StateStore[T any] struct {
	state     Store[State[T]]
	stateInit func(ctx context.Context) T
}

func NewStateStore[T any](
	stateCore Cache[State[T]],
	stateInit func(ctx context.Context) T,
) *StateStore[T] {
	return &StateStore[T]{
		state:     NewCache(stateCore, "agent:state", StateKeyFromContext),
		stateInit: stateInit,
	}
}

func NewMemoryStateStore[T any](stateInit func(ctx context.Context) T) *StateStore[T] {
	return NewStateStore[T](
		NewMemoryCore[State[T]](),
		stateInit,
	)
}

func (s *StateStore[T]) InitState(ctx context.Context) State[T] {
	if s.stateInit != nil {
		return State[T]{
			Phase:     types.PhaseCollecting,
			FormState: s.stateInit(ctx),
		}
	}
	var zero T
	return State[T]{
		Phase:     types.PhaseCollecting,
		FormState: zero,
	}
}

func (s *StateStore[T]) Load(ctx context.Context) (*State[T], error) {
	st, ok, err := s.state.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		st = s.InitState(ctx)
	}
	return &st, nil
}

func (s *StateStore[T]) Save(ctx context.Context, state *State[T]) error {
	if state == nil {
		return nil
	}
	if state.Phase == "" {
		state.Phase = types.PhaseCollecting
	}
	return s.state.Set(ctx, *state)
}

func (s *StateStore[T]) Clear(ctx context.Context) error {
	return s.state.Del(ctx)
}

var _ StateReadWriter[any] = (*StateStore[any])(nil)

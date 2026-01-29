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

type StateStore[T any] struct {
	store     Store[*State[T]]
	stateInit func(ctx context.Context) T
}

func NewStateStore[T any](
	store Store[*State[T]],
	stateInit func(ctx context.Context) T,
) *StateStore[T] {
	return &StateStore[T]{
		store:     store,
		stateInit: stateInit,
	}
}

func (s *StateStore[T]) initState(ctx context.Context) *State[T] {
	if s.stateInit != nil {
		return &State[T]{
			Phase:     types.PhaseCollecting,
			FormState: s.stateInit(ctx),
		}
	}
	var zero T
	return &State[T]{
		Phase:     types.PhaseCollecting,
		FormState: zero,
	}
}

func (s *StateStore[T]) Load(ctx context.Context) (*State[T], error) {
	st, ok, err := s.store.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		st = s.initState(ctx)
	}
	return st, nil
}

func (s *StateStore[T]) Save(ctx context.Context, state *State[T]) error {
	if state == nil {
		return nil
	}
	if state.Phase == "" {
		state.Phase = types.PhaseCollecting
	}
	return s.store.Set(ctx, state)
}

func (s *StateStore[T]) Clear(ctx context.Context) error {
	return s.store.Del(ctx)
}

var _ StateReadWriter[any] = (*StateStore[any])(nil)

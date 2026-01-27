package agent

import (
	"context"
	"sync"

	"github.com/tbxark/formagent/types"
)

// State represents the agent state stored outside the agent instance.
type State[T any] struct {
	Phase     types.Phase
	FormState T
}

// StateReadWriter provides read/write access to state using context for routing.
type StateReadWriter[T any] interface {
	Read(ctx context.Context) (State[T], error)
	Write(ctx context.Context, state State[T]) error
}

type stateKeyContext struct{}

const defaultStateKey = "default"

// WithStateKey sets a routing key for state storage in the context.
func WithStateKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, stateKeyContext{}, key)
}

// StateKeyFromContext gets the routing key from the context.
func StateKeyFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(stateKeyContext{})
	if value == nil {
		return "", false
	}
	key, ok := value.(string)
	return key, ok
}

func stateKeyOrDefault(ctx context.Context) string {
	key, ok := StateKeyFromContext(ctx)
	if ok && key != "" {
		return key
	}
	return defaultStateKey
}

// MemoryStateReadWriter is an in-memory implementation for testing and local usage.
type MemoryStateReadWriter[T any] struct {
	mu     sync.RWMutex
	states map[string]State[T]
}

func NewMemoryStateReadWriter[T any]() *MemoryStateReadWriter[T] {
	return &MemoryStateReadWriter[T]{
		states: make(map[string]State[T]),
	}
}

func (m *MemoryStateReadWriter[T]) Read(ctx context.Context) (State[T], error) {
	m.mu.RLock()
	state, ok := m.states[stateKeyOrDefault(ctx)]
	m.mu.RUnlock()
	if ok {
		return state, nil
	}

	var zero T
	return State[T]{
		Phase:     types.PhaseCollecting,
		FormState: zero,
	}, nil
}

func (m *MemoryStateReadWriter[T]) Write(ctx context.Context, state State[T]) error {
	if state.Phase == "" {
		state.Phase = types.PhaseCollecting
	}
	m.mu.Lock()
	m.states[stateKeyOrDefault(ctx)] = state
	m.mu.Unlock()
	return nil
}

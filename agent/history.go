package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

type Trimmer interface {
	Trim(history []*schema.Message) []*schema.Message
}

// KeepSystemLastNTrimmer keeps all system messages and the last N non-system messages.
// When N <= 0, it keeps only system messages.
type KeepSystemLastNTrimmer struct {
	N int
}

func (t KeepSystemLastNTrimmer) Trim(history []*schema.Message) []*schema.Message {
	if len(history) == 0 {
		return history
	}

	if t.N <= 0 {
		out := make([]*schema.Message, 0, len(history))
		for _, m := range history {
			if m != nil && m.Role == schema.System {
				out = append(out, m)
			}
		}
		return out
	}

	nonSystemIdx := make([]int, 0, len(history))
	for i, m := range history {
		if m == nil {
			continue
		}
		if m.Role != schema.System {
			nonSystemIdx = append(nonSystemIdx, i)
		}
	}
	if len(nonSystemIdx) <= t.N {
		return history
	}

	keep := make(map[int]struct{}, t.N)
	for _, i := range nonSystemIdx[len(nonSystemIdx)-t.N:] {
		keep[i] = struct{}{}
	}

	out := make([]*schema.Message, 0, len(history))
	for i, m := range history {
		if m == nil {
			continue
		}
		if m.Role == schema.System {
			out = append(out, m)
			continue
		}
		if _, ok := keep[i]; ok {
			out = append(out, m)
		}
	}
	return out
}

type HistoryReadWriter interface {
	Load(ctx context.Context) ([]*schema.Message, error)
	Save(ctx context.Context, history []*schema.Message) error
	Clear(ctx context.Context) error

	// Append loads history, appends msgs with de-duplication, trims, then saves.
	// It returns the saved history for convenient passing to adk.AgentInput.
	Append(ctx context.Context, msgs ...*schema.Message) ([]*schema.Message, error)
}

type HistoryStore struct {
	store   Store[[]*schema.Message]
	trimmer Trimmer
}

func NewHistoryStore(core Cache[[]*schema.Message], trimmer Trimmer) *HistoryStore {
	return &HistoryStore{
		store:   NewCache(core, "agent:history", StateKeyFromContext),
		trimmer: trimmer,
	}
}

func NewMemoryHistoryStore(trimmer Trimmer) *HistoryStore {
	return NewHistoryStore(NewMemoryCore[[]*schema.Message](), trimmer)
}

func (s *HistoryStore) Load(ctx context.Context) ([]*schema.Message, error) {
	hist, ok, err := s.store.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return hist, nil
}

func (s *HistoryStore) Save(ctx context.Context, history []*schema.Message) error {
	history = normalizeHistory(history)
	history = s.trim(history)
	return s.store.Set(ctx, history)
}

func (s *HistoryStore) Clear(ctx context.Context) error {
	return s.store.Del(ctx)
}

func (s *HistoryStore) Append(ctx context.Context, msgs ...*schema.Message) ([]*schema.Message, error) {
	hist, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	hist = appendHistory(hist, msgs...)
	if err := s.Save(ctx, hist); err != nil {
		return nil, err
	}
	return hist, nil
}

func (s *HistoryStore) trim(history []*schema.Message) []*schema.Message {
	if s == nil || s.trimmer == nil {
		return history
	}
	return s.trimmer.Trim(history)
}

func appendHistory(history []*schema.Message, msgs ...*schema.Message) []*schema.Message {
	if len(msgs) == 0 {
		return history
	}
	out := history
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		if len(out) > 0 {
			last := out[len(out)-1]
			if last != nil && last.Role == msg.Role && last.Content == msg.Content {
				continue
			}
		}
		out = append(out, msg)
	}
	return out
}

func normalizeHistory(history []*schema.Message) []*schema.Message {
	if len(history) == 0 {
		return history
	}
	out := make([]*schema.Message, 0, len(history))
	for _, m := range history {
		if m != nil {
			out = append(out, m)
		}
	}
	return out
}

var _ HistoryReadWriter = (*HistoryStore)(nil)

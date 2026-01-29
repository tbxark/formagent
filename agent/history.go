package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

type HistoryReadWriter interface {
	Load(ctx context.Context) ([]*schema.Message, error)
	Save(ctx context.Context, history []*schema.Message) error
	Clear(ctx context.Context) error
	Append(ctx context.Context, messages ...*schema.Message) ([]*schema.Message, error)
}

type HistoryStore struct {
	store Store[[]*schema.Message]
}

func NewHistoryStore(store Store[[]*schema.Message]) *HistoryStore {
	return &HistoryStore{
		store: store,
	}
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
	return s.store.Set(ctx, history)
}

func (s *HistoryStore) Clear(ctx context.Context) error {
	return s.store.Del(ctx)
}

func (s *HistoryStore) Append(ctx context.Context, messages ...*schema.Message) ([]*schema.Message, error) {
	hist, err := s.Load(ctx)
	if err != nil {
		return nil, err
	}
	hist = appendHistory(hist, messages...)
	if err := s.Save(ctx, hist); err != nil {
		return nil, err
	}
	return hist, nil
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

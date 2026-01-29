package agent

import (
	"context"
	"errors"
)

type KeyGen func(ctx context.Context) (string, bool)

type Store[S any] struct {
	core      Cache[S]
	namespace string
	keygen    KeyGen
}

func NewStore[S any](core Cache[S], namespace string, keygen KeyGen) Store[S] {
	return Store[S]{
		core:      core,
		namespace: namespace,
		keygen:    keygen,
	}
}

func (c Store[S]) key(ctx context.Context) (string, bool) {
	key, exist := c.keygen(ctx)
	if !exist {
		return "", false
	}
	return c.namespace + ":" + key, true
}

func (c Store[S]) Set(ctx context.Context, val S) error {
	key, ok := c.key(ctx)
	if !ok {
		return errors.New("key not found")
	}
	return c.core.Set(ctx, key, val)
}

func (c Store[S]) Get(ctx context.Context) (S, bool, error) {
	key, ok := c.key(ctx)
	if !ok {
		var zero S
		return zero, false, errors.New("key not found")
	}
	return c.core.Get(ctx, key)
}

func (c Store[S]) Del(ctx context.Context) error {
	key, ok := c.key(ctx)
	if !ok {
		return errors.New("key not found")
	}
	return c.core.Del(ctx, key)
}

func (c Store[S]) Exists(ctx context.Context) (bool, error) {
	key, ok := c.key(ctx)
	if !ok {
		return false, errors.New("key not found")
	}
	return c.core.Exists(ctx, key)
}

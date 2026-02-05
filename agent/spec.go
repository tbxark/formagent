package agent

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type FormSpec[T any] interface {
	Summary(ctx context.Context, current T) string
	MissingFacts(ctx context.Context, current T) []types.FieldInfo
	ValidateFacts(ctx context.Context, current T) []types.FieldInfo
}

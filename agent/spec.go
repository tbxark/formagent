package agent

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type FormSpec[T any] interface {
	JsonSchema() (string, error)

	MissingFacts(current T) []types.FieldInfo
	ValidateFacts(current T) []types.FieldInfo

	Summary(current T) string
}

type FormManager[T any] interface {
	Cancel(ctx context.Context, form T) error
	Submit(ctx context.Context, form T) error
}

package dialogue

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type Request[T any] struct {
	CurrentState T
	Phase        types.Phase

	MissingFields    []types.FieldInfo
	ValidationErrors []types.FieldInfo
}

type Generator[T any] interface {
	GenerateDialogue(ctx context.Context, req *Request[T]) (string, error)
	GenerateDialogueStream(ctx context.Context, req *Request[T]) (*schema.StreamReader[string], error)
}

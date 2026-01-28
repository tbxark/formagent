package dialogue

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type Generator[T any] interface {
	GenerateDialogue(ctx context.Context, req *types.ToolRequest[T]) (string, error)
	GenerateDialogueStream(ctx context.Context, req *types.ToolRequest[T]) (*schema.StreamReader[string], error)
}

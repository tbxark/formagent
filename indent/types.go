package indent

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type Intent string

const (
	Cancel    Intent = "cancel"
	Confirm   Intent = "confirm"
	Edit      Intent = "edit"
	DoNothing Intent = "do_nothing"
)

type Recognizer[T any] interface {
	RecognizerIntent(ctx context.Context, req *types.ToolRequest[T]) (Intent, error)
}

package dialogue

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type NextTurnPlan struct {
	Message         string `json:"message"`
	SuggestedAction string `json:"suggested_action,omitempty"`
}

type Request[T any] struct {
	CurrentState T
	Phase        types.Phase

	MissingFields    []types.FieldInfo
	ValidationErrors []types.ValidationError

	LastUserInput string
	PatchApplied  bool
}

type Generator[T any] interface {
	GenerateDialogue(ctx context.Context, req Request[T]) (*NextTurnPlan, error)
}

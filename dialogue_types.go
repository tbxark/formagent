package formagent

import "context"

type NextTurnPlan struct {
	Message         string `json:"message"`
	SuggestedAction string `json:"suggested_action,omitempty"`
}

type DialogueRequest[T any] struct {
	CurrentState T
	Phase        Phase

	MissingFields    []FieldInfo
	ValidationErrors []ValidationError

	LastUserInput string
	PatchApplied  bool
}

type DialogueGenerator[T any] interface {
	GenerateDialogue(ctx context.Context, req DialogueRequest[T]) (*NextTurnPlan, error)
}

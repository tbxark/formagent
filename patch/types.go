package patch

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

type UpdateFormArgs struct {
	Ops []PatchOperation `json:"ops"`
}

type PatchRequest[T any] struct {
	UserInput     string
	CurrentState  T
	AllowedPaths  []string
	MissingFields []types.FieldInfo

	FieldGuidance map[string]string
}

type PatchGenerator[T any] interface {
	GeneratePatch(ctx context.Context, req PatchRequest[T]) (UpdateFormArgs, error)
}

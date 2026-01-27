package patch

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type Operation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

type UpdateFormArgs struct {
	Ops []Operation `json:"ops"`
}

type Request[T any] struct {
	UserInput     string
	CurrentState  T
	AllowedPaths  []string
	MissingFields []types.FieldInfo

	FieldGuidance map[string]string
}

type Generator[T any] interface {
	GeneratePatch(ctx context.Context, req Request[T]) (UpdateFormArgs, error)
}

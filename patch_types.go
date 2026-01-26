package formagent

import "context"

type PatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

type UpdateFormArgs struct {
	Ops []PatchOperation `json:"ops"`
}

type PatchGenerator[T any] interface {
	GeneratePatch(ctx context.Context, req PatchRequest[T]) (UpdateFormArgs, error)
}

type PatchRequest[T any] struct {
	UserInput string

	CurrentState T

	AllowedPaths []string

	MissingFields []FieldInfo

	FieldGuidance map[string]string
}

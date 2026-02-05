package patch

import (
	"context"

	"github.com/tbxark/formagent/types"
)

const (
	OperationAdd     = "add"
	OperationReplace = "replace"
	OperationRemove  = "remove"
)

type Operation struct {
	Op          string `json:"op" jsonschema:"enum=add,enum=replace,enum=remove,description=RFC6902 operation type (add, replace, remove)"`
	Path        string `json:"path" jsonschema:"pattern=^/.*$,description=RFC6902 JSON Pointer, must start with '/'"`
	Value       any    `json:"value,omitempty" jsonschema:"description=Value to apply for add/replace operations (optional for remove)"`
	Description string `json:"description,omitempty" jsonschema:"description=Description of the operation, Defaults to empty string"`
}

type UpdateFormArgs struct {
	Ops []Operation `json:"ops" jsonschema:"description=Array of RFC6902 JSON Patch operations to update the form"`
}

type Generator[T any] interface {
	GeneratePatch(ctx context.Context, req *types.ToolRequest[T]) (*UpdateFormArgs, error)
}

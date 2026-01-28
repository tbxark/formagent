package agent

import (
	"context"

	"github.com/tbxark/formagent/types"
)

// FormSpec 定义表单规范接口，需要为每个表单类型实现
type FormSpec[T any] interface {
	JsonSchema() (string, error)

	MissingFacts(current T) []types.FieldInfo
	ValidateFacts(current T) []types.FieldInfo

	Summary(current T) string
	Submit(ctx context.Context, final T) error
}

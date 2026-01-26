package formagent

import "context"

// FormSpec 定义表单规范接口，需要为每个表单类型实现
type FormSpec[T any] interface {
	AllowedJSONPointers() []string
	FieldGuide(fieldPath string) string
	MissingFacts(current T) []FieldInfo
	ValidateFacts(current T) []ValidationError
	Summary(current T) string
	Submit(ctx context.Context, final T) error
}

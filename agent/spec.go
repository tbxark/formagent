package agent

import (
	"github.com/tbxark/formagent/types"
)

type FormSpec[T any] interface {
	JsonSchema() (string, error)

	MissingFacts(current T) []types.FieldInfo
	ValidateFacts(current T) []types.FieldInfo

	Summary(current T) string
}

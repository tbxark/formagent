package agent

import "github.com/tbxark/formagent/types"

type Request struct {
	UserInput string `json:"user_input" jsonschema:"description=The input provided by the user to the agent"`
}
type Response[T any] struct {
	Message   string            `json:"message"`
	Phase     types.Phase       `json:"phase"`
	FormState T                 `json:"form_state"`
	Completed bool              `json:"completed"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

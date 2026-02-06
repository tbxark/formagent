package agent

import (
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type State[T any] struct {
	Phase     types.Phase `json:"phase" jsonschema:"enum=collecting,enum=confirming,enum=submitted,enum=cancelled,description=The current phase of the form filling process"`
	FormState T           `json:"form_state" jsonschema:"description=The current state of the form being filled"`
}
type Request[T any] struct {
	State       *State[T]         `json:"state"`
	ChatHistory []*schema.Message `json:"chat_history"`
}
type Response[T any] struct {
	Message  string            `json:"message,omitempty"`
	State    *State[T]         `json:"state,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type StreamResponse[T any] struct {
	MessageStream *schema.StreamReader[string] `json:"-"`
	State         *State[T]                    `json:"state,omitempty"`
	Metadata      map[string]string            `json:"metadata,omitempty"`
}

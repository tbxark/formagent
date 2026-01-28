package agent

import (
	"github.com/cloudwego/eino/schema"
	"github.com/tbxark/formagent/types"
)

type State[T any] struct {
	Phase          types.Phase `json:"phase" jsonschema:"enum=collecting,enum=confirming,enum=submitted,enum=cancelled,description=The current phase of the form filling process"`
	LatestQuestion string      `json:"latest_question" jsonschema:"description=The latest question asked to the user"`
	FormState      T           `json:"form_state" jsonschema:"description=The current state of the form being filled"`
}
type Request[T any] struct {
	State       *State[T]         `json:"state" jsonschema:"description=The current state of the form filling process"`
	ChatHistory []*schema.Message `json:"chat_history" jsonschema:"description=The chat history between the user and the agent"`
}
type Response[T any] struct {
	Message  string            `json:"message"`
	State    *State[T]         `json:"state"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

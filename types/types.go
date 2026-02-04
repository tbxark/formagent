package types

import "github.com/cloudwego/eino/schema"

type Phase string

const (
	PhaseCollecting Phase = "collecting"
	PhaseConfirmed  Phase = "confirmed"
	PhaseCancelled  Phase = "cancelled"
)

type FieldInfo struct {
	JSONPointer string `json:"json_pointer"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type ToolRequest[T any] struct {
	State        T
	StateSummary string

	Phase    Phase
	Messages []*schema.Message

	MissingFields    []FieldInfo
	ValidationErrors []FieldInfo
}

package types

type Phase string

const (
	PhaseCollecting Phase = "collecting"
	PhaseConfirming Phase = "confirming"
	PhaseConfirmed  Phase = "confirmed"
	PhaseCancelled  Phase = "cancelled"
)

type FieldInfo struct {
	JSONPointer string `json:"json_pointer"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

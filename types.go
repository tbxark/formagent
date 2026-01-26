package formagent

import (
	"time"
)

type Phase string

const (
	PhaseCollecting Phase = "collecting"
	PhaseConfirming Phase = "confirming"
	PhaseSubmitted  Phase = "submitted"
	PhaseCancelled  Phase = "cancelled"
)

type FieldInfo struct {
	JSONPointer string `json:"json_pointer"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

type ValidationError struct {
	JSONPointer string `json:"json_pointer"`
	Message     string `json:"message"`
}

type Response[T any] struct {
	Message   string            `json:"message"`
	Phase     Phase             `json:"phase"`
	FormState T                 `json:"form_state"`
	Completed bool              `json:"completed"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Checkpoint[T any] struct {
	Version      string            `json:"version"`
	Phase        Phase             `json:"phase"`
	FormState    T                 `json:"form_state"`
	Timestamp    time.Time         `json:"timestamp"`
	AllowedPaths []string          `json:"allowed_paths"`
	Missing      []FieldInfo       `json:"missing,omitempty"`
	Issues       []ValidationError `json:"issues,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	LastUserText string            `json:"last_user_text,omitempty"`
}

package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

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

type MessagePair struct {
	Question string `json:"question,omitempty"`
	Answer   string `json:"answer,omitempty"`
}

type ToolRequest[T any] struct {
	State       T
	StateSchema string
	Phase       Phase
	MessagePair MessagePair

	MissingFields    []FieldInfo
	ValidationErrors []FieldInfo
}

func (req ToolRequest[T]) ToPromptMessage() (string, error) {
	stateJSON, err := json.Marshal(req.State)
	if err != nil {
		return "", err
	}
	sections := []string{
		fmt.Sprintf("# Current Date: \n %s", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("# Form state JSON:\n```json\n%s\n```", string(stateJSON)),
	}
	if req.StateSchema != "" {
		sections = append(sections, fmt.Sprintf("# Form state schema JSON:\n```json\n%s\n```", req.StateSchema))
	}
	if req.Phase != "" {
		sections = append(sections, fmt.Sprintf("# Current Phase:\n%s", req.Phase))
	}
	if req.MessagePair.Question != "" || req.MessagePair.Answer != "" {
		sections = append(sections, "# Latest Dialogue:")
		if req.MessagePair.Question != "" {
			sections = append(sections, fmt.Sprintf("## Assistant Question:\n%s", req.MessagePair.Question))
		}
		if req.MessagePair.Answer != "" {
			sections = append(sections, fmt.Sprintf("## User Answer:\n%s", req.MessagePair.Answer))
		}
	}
	if s := formatMissingFieldsSectionForDialogue(req.MissingFields); s != "" {
		sections = append(sections, s)
	}
	if s := formatValidationErrorsSection(req.ValidationErrors); s != "" {
		sections = append(sections, s)
	}
	return strings.Join(sections, "\n\n"), nil
}

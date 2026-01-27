package dialogue

import (
	"fmt"

	"github.com/tbxark/formagent/types"
)

func FormatUserInputSection(lastInput string, patchApplied bool) string {
	if lastInput == "" {
		return ""
	}
	return fmt.Sprintf("User's last input: %s\nInformation extracted: %v\n", lastInput, patchApplied)
}

func FormatMissingFieldsSectionForDialogue(fields []types.FieldInfo, phase types.Phase) string {
	if len(fields) == 0 {
		if phase == types.PhaseCollecting {
			return "All required fields have been filled.\n"
		}
		return ""
	}
	result := "Missing required fields:\n"
	for _, field := range fields {
		result += fmt.Sprintf("  - %s (%s)", field.DisplayName, field.JSONPointer)
		if field.Description != "" {
			result += fmt.Sprintf(": %s", field.Description)
		}
		result += "\n"
	}
	return result
}

func FormatValidationErrorsSection(errors []types.ValidationError) string {
	if len(errors) == 0 {
		return ""
	}
	result := "Validation errors:\n"
	for _, err := range errors {
		result += fmt.Sprintf("  - %s: %s\n", err.JSONPointer, err.Message)
	}
	return result
}

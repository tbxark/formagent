package dialogue

import (
	"fmt"
	"strings"

	"github.com/tbxark/formagent/types"
)

func formatUserInputSection(lastInput string, patchApplied bool) string {
	if lastInput == "" {
		return ""
	}
	extracted := "no"
	if patchApplied {
		extracted = "yes"
	}
	return fmt.Sprintf("# User input:\n%s\n> extracted info: %s", lastInput, extracted)
}

func formatMissingFieldsSectionForDialogue(fields []types.FieldInfo, phase types.Phase) string {
	if len(fields) == 0 {
		if phase == types.PhaseCollecting {
			return "# Missing required fields:\n none"
		}
		return ""
	}
	result := "# Missing required fields:\n"
	for _, field := range fields {
		result += fmt.Sprintf("- %s [%s]", field.DisplayName, field.JSONPointer)
		if field.Description != "" {
			result += fmt.Sprintf(": %s", field.Description)
		}
		result += "\n"
	}
	return strings.TrimRight(result, "\n")
}

func formatValidationErrorsSection(errors []types.ValidationError) string {
	if len(errors) == 0 {
		return ""
	}
	result := "# Validation errors:\n"
	for _, err := range errors {
		result += fmt.Sprintf("- %s: %s\n", err.JSONPointer, err.Message)
	}
	return strings.TrimRight(result, "\n")
}

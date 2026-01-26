package formagent

import (
	"fmt"
	"strings"
)

func formatAllowedPaths(paths []string) string {
	if len(paths) == 0 {
		return "  (all paths allowed)"
	}
	var sb strings.Builder
	for _, path := range paths {
		sb.WriteString("  - ")
		sb.WriteString(path)
		sb.WriteString("\n")
	}
	return sb.String()
}

func formatMissingFieldsSection(fields []FieldInfo) string {
	if len(fields) == 0 {
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

func formatFieldGuidanceSection(guidance map[string]string) string {
	if len(guidance) == 0 {
		return ""
	}
	result := "Field guidance:\n"
	for path, guide := range guidance {
		result += fmt.Sprintf("  - %s: %s\n", path, guide)
	}
	return result
}

func formatUserInputSection(lastInput string, patchApplied bool) string {
	if lastInput == "" {
		return ""
	}
	return fmt.Sprintf("User's last input: %s\nInformation extracted: %v\n", lastInput, patchApplied)
}

func formatMissingFieldsSectionForDialogue(fields []FieldInfo, phase Phase) string {
	if len(fields) == 0 {
		if phase == PhaseCollecting {
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

func formatValidationErrorsSection(errors []ValidationError) string {
	if len(errors) == 0 {
		return ""
	}
	result := "Validation errors:\n"
	for _, err := range errors {
		result += fmt.Sprintf("  - %s: %s\n", err.JSONPointer, err.Message)
	}
	return result
}

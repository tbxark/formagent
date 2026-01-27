package patch

import (
	"fmt"
	"strings"

	"github.com/tbxark/formagent/types"
)

func FormatAllowedPaths(paths []string) string {
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

func FormatMissingFieldsSection(fields []types.FieldInfo) string {
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

func FormatFieldGuidanceSection(guidance map[string]string) string {
	if len(guidance) == 0 {
		return ""
	}
	result := "Field guidance:\n"
	for path, guide := range guidance {
		result += fmt.Sprintf("  - %s: %s\n", path, guide)
	}
	return result
}

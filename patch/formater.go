package patch

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tbxark/formagent/types"
)

func formatAllowedPaths(paths []string) string {
	if len(paths) == 0 {
		return "all (no restriction)"
	}
	var sb strings.Builder
	for _, path := range paths {
		sb.WriteString("- ")
		sb.WriteString(path)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatMissingFieldsSection(fields []types.FieldInfo) string {
	if len(fields) == 0 {
		return ""
	}
	result := "Missing required fields:\n"
	for _, field := range fields {
		result += fmt.Sprintf("- %s [%s]", field.DisplayName, field.JSONPointer)
		if field.Description != "" {
			result += fmt.Sprintf(": %s", field.Description)
		}
		result += "\n"
	}
	return strings.TrimRight(result, "\n")
}

func formatFieldGuidanceSection(guidance map[string]string) string {
	if len(guidance) == 0 {
		return ""
	}
	keys := make([]string, 0, len(guidance))
	for path := range guidance {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	result := "Field guidance:\n"
	for _, path := range keys {
		result += fmt.Sprintf("- %s: %s\n", path, guidance[path])
	}
	return strings.TrimRight(result, "\n")
}

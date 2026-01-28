package dialogue

import (
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/tbxark/formagent/types"
)

func formatMissingFieldsSectionForDialogue(fields []types.FieldInfo, phase types.Phase) string {
	if len(fields) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Missing required fields:\n")
	table := tablewriter.NewTable(&buf, tablewriter.WithRenderer(renderer.NewMarkdown()))
	table.Header("Field", "Pointer", "Description")
	for _, field := range fields {
		table.Append(field.DisplayName, field.JSONPointer, field.Description)
	}
	table.Render()
	return buf.String()
}

func formatValidationErrorsSection(errors []types.FieldInfo) string {
	if len(errors) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Validation errors:\n")
	table := tablewriter.NewTable(&buf, tablewriter.WithRenderer(renderer.NewMarkdown()))
	table.Header("Pointer", "Error")
	for _, err := range errors {
		table.Append(err.JSONPointer, err.Description)
	}
	table.Render()
	return buf.String()
}

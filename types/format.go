package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
)

func formatMissingFieldsSectionForDialogue(fields []FieldInfo) string {
	if len(fields) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Missing required fields:\n")
	table := tablewriter.NewTable(&buf, tablewriter.WithRenderer(renderer.NewMarkdown()))
	table.Header("Field", "Pointer", "Description")
	for _, field := range fields {
		_ = table.Append(field.DisplayName, field.JSONPointer, field.Description)
	}
	_ = table.Render()
	return buf.String()
}

func formatValidationErrorsSection(errors []FieldInfo) string {
	if len(errors) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Validation errors:\n")
	table := tablewriter.NewTable(&buf, tablewriter.WithRenderer(renderer.NewMarkdown()))
	table.Header("Pointer", "Error")
	for _, err := range errors {
		_ = table.Append(err.JSONPointer, err.Description)
	}
	_ = table.Render()
	return buf.String()
}

func formatMessageHistory(messages []*schema.Message) string {
	if len(messages) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Dialogue history:\n")
	table := tablewriter.NewTable(&buf, tablewriter.WithRenderer(renderer.NewMarkdown()))
	table.Header("Role", "Content")
	for _, msg := range messages {
		_ = table.Append(msg.Role, msg.Content)
	}
	_ = table.Render()
	return buf.String()
}

func FormatToolRequest[T any](req *ToolRequest[T]) (string, error) {
	stateJSON, err := json.Marshal(req.State)
	if err != nil {
		return "", err
	}
	sections := []string{
		fmt.Sprintf("# Current Date: \n %s", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("# Form state JSON:\n```json\n%s\n```", string(stateJSON)),
	}
	if req.StateSummary != "" {
		sections = append(sections, fmt.Sprintf("# Form state summary:\n%s", req.StateSummary))
	}
	if req.StateSchema != "" {
		sections = append(sections, fmt.Sprintf("# Form state schema JSON:\n```json\n%s\n```", req.StateSchema))
	}
	if req.Phase != "" {
		sections = append(sections, fmt.Sprintf("# Current Phase:\n%s", req.Phase))
	}
	if s := formatMessageHistory(req.Messages); s != "" {
		sections = append(sections, s)
	}
	if s := formatMissingFieldsSectionForDialogue(req.MissingFields); s != "" {
		sections = append(sections, s)
	}
	if s := formatValidationErrorsSection(req.ValidationErrors); s != "" {
		sections = append(sections, s)
	}
	return strings.Join(sections, "\n\n"), nil
}

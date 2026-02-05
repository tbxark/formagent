package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

func FormatMissingFieldsSectionForDialogue(fields []FieldInfo) string {
	if len(fields) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Missing required fields:\n")
	for _, field := range fields {
		buf.WriteString("- ")
		if field.DisplayName != "" {
			buf.WriteString(field.DisplayName)
			if field.JSONPointer != "" {
				buf.WriteString(" (`")
				buf.WriteString(field.JSONPointer)
				buf.WriteString("`)")
			}
		} else if field.JSONPointer != "" {
			buf.WriteString("`")
			buf.WriteString(field.JSONPointer)
			buf.WriteString("`")
		} else {
			buf.WriteString("(unnamed)")
		}
		if field.Description != "" {
			buf.WriteString(": ")
			buf.WriteString(field.Description)
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func FormatValidationErrorsSection(errors []FieldInfo) string {
	if len(errors) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("# Validation errors:\n")
	for _, err := range errors {
		buf.WriteString("- ")
		if err.JSONPointer != "" {
			buf.WriteString("`")
			buf.WriteString(err.JSONPointer)
			buf.WriteString("`")
		} else {
			buf.WriteString("(unknown)")
		}
		if err.Description != "" {
			buf.WriteString(": ")
			buf.WriteString(err.Description)
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func FormatMessageHistory(messages []*schema.Message) string {
	if len(messages) == 0 {
		return ""
	}
	lastUserIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if string(messages[i].Role) == "user" {
			lastUserIndex = i
			break
		}
	}
	var history strings.Builder
	for i, msg := range messages {
		if i == lastUserIndex {
			continue
		}
		history.WriteString("- ")
		if msg.Role != "" {
			history.WriteString(string(msg.Role))
		} else {
			history.WriteString("unknown")
		}
		if msg.Content != "" {
			history.WriteString(": ")
			history.WriteString(msg.Content)
		} else {
			history.WriteString(": (empty)")
		}
		history.WriteString("\n")
	}
	var buf strings.Builder
	if history.Len() > 0 {
		buf.WriteString("# Dialogue history:\n")
		buf.WriteString(history.String())
	}
	if lastUserIndex != -1 {
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("# Latest user message:\n")
		content := messages[lastUserIndex].Content
		if content == "" {
			buf.WriteString("> (empty)")
			return buf.String()
		}
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			buf.WriteString("> ")
			if line != "" {
				buf.WriteString(line)
			}
			if i != len(lines)-1 {
				buf.WriteString("\n")
			}
		}
	}
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
	if req.Phase != "" {
		sections = append(sections, fmt.Sprintf("# Current Phase:\n%s", req.Phase))
	}
	if s := FormatMessageHistory(req.Messages); s != "" {
		sections = append(sections, s)
	}
	if s := FormatMissingFieldsSectionForDialogue(req.MissingFields); s != "" {
		sections = append(sections, s)
	}
	if s := FormatValidationErrorsSection(req.ValidationErrors); s != "" {
		sections = append(sections, s)
	}
	return strings.Join(sections, "\n\n"), nil
}

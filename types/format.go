package types

import (
	"fmt"
	"regexp"
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
		history.WriteString("#### ")
		if msg.Role != "" {
			history.WriteString(string(msg.Role))
		} else {
			history.WriteString("unknown")
		}
		history.WriteString(": \n")
		content := msg.Content
		if content == "" {
			content = "(empty)"
		}
		history.WriteString(WrapMarkdownCodeBlock(content, ""))
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
			content = "(empty)"
		}
		buf.WriteString(WrapMarkdownCodeBlock(content, ""))
		return buf.String()
	}
	return buf.String()
}

func FormatToolRequest[T any](req *ToolRequest[T]) (string, error) {
	sections := []string{
		fmt.Sprintf("# Current Date: \n %s", time.Now().Format(time.RFC3339)),
	}
	if req.StateSummary != "" {
		sections = append(sections, fmt.Sprintf("# Form state:\n%s", req.StateSummary))
	}
	if req.Phase != "" {
		sections = append(sections, fmt.Sprintf("# Current Phase:\n**%s**", req.Phase))
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

func WrapMarkdownCodeBlock(text, lang string) string {
	re := regexp.MustCompile("`+")
	longest := 0
	for _, m := range re.FindAllString(text, -1) {
		if len(m) > longest {
			longest = len(m)
		}
	}
	fenceLen := longest + 1
	if fenceLen < 3 {
		fenceLen = 3
	}
	fence := strings.Repeat("`", fenceLen)
	var sb strings.Builder
	sb.WriteString(fence)
	if lang != "" {
		sb.WriteString(lang)
	}
	sb.WriteString("\n")
	sb.WriteString(text)
	sb.WriteString("\n")
	sb.WriteString(fence)
	return sb.String()
}

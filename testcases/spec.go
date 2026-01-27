package testcases

import (
	"context"
	"fmt"

	"github.com/tbxark/formagent/types"
)

type UserRegistrationForm struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

type FormSpec struct{}

func (s *FormSpec) AllowedJSONPointers() []string {
	return []string{"/name", "/email", "/age", "/password"}
}

func (s *FormSpec) FieldGuide(fieldPath string) string {
	guides := map[string]string{
		"/email": "请提供有效的电子邮件地址",
		"/age":   "年龄必须在 18-100 之间",
	}
	return guides[fieldPath]
}

func (s *FormSpec) MissingFacts(current UserRegistrationForm) []types.FieldInfo {
	var missing []types.FieldInfo
	if current.Name == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/name",
			DisplayName: "姓名",
			Required:    true,
		})
	}
	if current.Email == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/email",
			DisplayName: "邮箱",
			Required:    true,
		})
	}
	return missing
}

func (s *FormSpec) ValidateFacts(current UserRegistrationForm) []types.ValidationError {
	var errors []types.ValidationError
	if current.Age < 18 || current.Age > 100 {
		errors = append(errors, types.ValidationError{
			JSONPointer: "/age",
			Message:     "年龄必须在 18-100 之间",
		})
	}
	return errors
}

func (s *FormSpec) Summary(current UserRegistrationForm) string {
	return fmt.Sprintf("姓名: %s, 邮箱: %s, 年龄: %d", current.Name, current.Email, current.Age)
}

func (s *FormSpec) Submit(ctx context.Context, final UserRegistrationForm) error {
	fmt.Printf("提交表单: %+v\n", final)
	return nil
}

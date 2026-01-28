package main

import (
	"encoding/json"
	"fmt"

	"github.com/eino-contrib/jsonschema"
	"github.com/tbxark/formagent/agent"
	"github.com/tbxark/formagent/types"
)

type Invoice struct {
	Title       string  `json:"title" jsonschema:"description=报销抬头"`
	Amount      float64 `json:"amount" jsonschema:"description=金额"`
	Date        string  `json:"date" jsonschema:"description=日期，格式为 YYYY-MM-DD"`
	Category    string  `json:"category" jsonschema:"description=类别"`
	Payee       string  `json:"payee" jsonschema:"description=收款人"`
	Description string  `json:"description" jsonschema:"description=备注"`
}

var _ agent.FormSpec[*Invoice] = (*InvoiceFormSpec)(nil)

type InvoiceFormSpec struct {
}

func (InvoiceFormSpec) JsonSchema() (string, error) {
	schema := jsonschema.Reflect(&Invoice{})
	schema.Title = "报销单"
	schema.Description = "用于提交报销申请的表单，包含报销抬头、金额、日期、类别、收款人和备注等字段。"
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON schema: %w", err)
	}
	return string(schemaBytes), nil
}

func (InvoiceFormSpec) MissingFacts(current *Invoice) []types.FieldInfo {
	var missing []types.FieldInfo
	if current.Title == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/title",
			DisplayName: "报销抬头",
			Required:    true,
		})
	}
	if current.Amount <= 0 {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/amount",
			DisplayName: "金额",
			Required:    true,
		})
	}
	if current.Date == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/date",
			DisplayName: "日期",
			Required:    true,
		})
	}
	if current.Category == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/category",
			DisplayName: "类别",
			Required:    true,
		})
	}
	if current.Payee == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/payee",
			DisplayName: "收款人",
			Required:    true,
		})
	}
	// Description 可选
	return missing
}

func (InvoiceFormSpec) ValidateFacts(current *Invoice) []types.FieldInfo {
	var errs []types.FieldInfo
	if current.Amount < 0 {
		errs = append(errs, types.FieldInfo{
			JSONPointer: "/amount",
			Description: "金额不能为负数",
		})
	}
	// 简单日期格式校验
	if len(current.Date) != 10 {
		errs = append(errs, types.FieldInfo{
			JSONPointer: "/date",
			Description: "日期格式应为 YYYY-MM-DD",
		})
	}
	return errs
}

func (InvoiceFormSpec) Summary(current *Invoice) string {
	return fmt.Sprintf("报销单摘要：\n抬头：%s\n金额：%.2f 元\n日期：%s\n类别：%s\n收款人：%s\n备注：%s",
		current.Title, current.Amount, current.Date, current.Category, current.Payee, current.Description)
}

package main

import (
	"context"
	"fmt"

	"github.com/tbxark/formagent/agent"
	"github.com/tbxark/formagent/types"
)

type Invoice struct {
	Title       string  `json:"Title"`       // 抬头
	Amount      float64 `json:"Amount"`      // 金额
	Date        string  `json:"Date"`        // 日期，格式如 2026-01-28
	Category    string  `json:"Category"`    // 类别，如“差旅”、“办公用品”
	Payee       string  `json:"Payee"`       // 收款人
	Description string  `json:"Description"` // 备注
}

var _ agent.FormSpec[Invoice] = (*InvoiceFormSpec)(nil)

type InvoiceFormSpec struct {
}

func (InvoiceFormSpec) AllowedJSONPointers() []string {
	return []string{
		"/Title",
		"/Amount",
		"/Date",
		"/Category",
		"/Payee",
		"/Description",
	}
}

func (InvoiceFormSpec) FieldGuide(fieldPath string) string {
	switch fieldPath {
	case "/Title":
		return "报销抬头，如公司名称或个人姓名"
	case "/Amount":
		return "报销金额，单位为元"
	case "/Date":
		return "报销日期，格式如 2026-01-28"
	case "/Category":
		return "报销类别，如‘差旅’、‘办公用品’等"
	case "/Payee":
		return "收款人姓名"
	case "/Description":
		return "备注信息，可选"
	default:
		return ""
	}
}

func (InvoiceFormSpec) MissingFacts(current Invoice) []types.FieldInfo {
	var missing []types.FieldInfo
	if current.Title == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/Title",
			DisplayName: "报销抬头",
			Required:    true,
		})
	}
	if current.Amount <= 0 {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/Amount",
			DisplayName: "金额",
			Required:    true,
		})
	}
	if current.Date == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/Date",
			DisplayName: "日期",
			Required:    true,
		})
	}
	if current.Category == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/Category",
			DisplayName: "类别",
			Required:    true,
		})
	}
	if current.Payee == "" {
		missing = append(missing, types.FieldInfo{
			JSONPointer: "/Payee",
			DisplayName: "收款人",
			Required:    true,
		})
	}
	// Description 可选
	return missing
}

func (InvoiceFormSpec) ValidateFacts(current Invoice) []types.ValidationError {
	var errs []types.ValidationError
	if current.Amount < 0 {
		errs = append(errs, types.ValidationError{
			JSONPointer: "/Amount",
			Message:     "金额不能为负数",
		})
	}
	// 简单日期格式校验
	if len(current.Date) != 10 {
		errs = append(errs, types.ValidationError{
			JSONPointer: "/Date",
			Message:     "日期格式应为 YYYY-MM-DD",
		})
	}
	return errs
}

func (InvoiceFormSpec) Summary(current Invoice) string {
	return fmt.Sprintf("报销单摘要：\n抬头：%s\n金额：%.2f 元\n日期：%s\n类别：%s\n收款人：%s\n备注：%s",
		current.Title, current.Amount, current.Date, current.Category, current.Payee, current.Description)
}

func (InvoiceFormSpec) Submit(ctx context.Context, final Invoice) error {
	// 实际业务可写入数据库等，这里仅打印
	fmt.Printf("已提交报销单: %+v\n", final)
	return nil
}

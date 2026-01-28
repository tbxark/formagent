package main

import (
	"context"
	"log/slog"

	"github.com/tbxark/formagent/agent"
)

var _ agent.FormManager[*Invoice] = (*InvoiceFormManager)(nil)

type InvoiceFormManager struct {
}

func (i *InvoiceFormManager) Cancel(ctx context.Context, form *Invoice) error {
	slog.Debug("Invoice form cancelled", "form", form)
	return nil
}

func (i *InvoiceFormManager) Submit(ctx context.Context, form *Invoice) error {
	slog.Info("Invoice form submitted", "form", form)
	return nil
}

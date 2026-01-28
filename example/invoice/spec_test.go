package main

import "testing"

func TestInvoiceFormSpec_JsonSchema(t *testing.T) {
	t.Log(InvoiceFormSpec{}.JsonSchema())
}

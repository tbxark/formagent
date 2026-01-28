package command

import (
	"context"

	"github.com/tbxark/formagent/types"
)

type Command string

const (
	Cancel    Command = "cancel"
	Confirm   Command = "confirm"
	Edit      Command = "edit"
	DoNothing Command = "do_nothing"
)

type Parser[T any] interface {
	ParseCommand(ctx context.Context, req *types.ToolRequest[T]) (Command, error)
}

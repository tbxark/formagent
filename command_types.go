package formagent

import "context"

type Command string

const (
	CommandCancel  Command = "cancel"
	CommandConfirm Command = "confirm"
	CommandBack    Command = "back"
	CommandNone    Command = ""
)

type CommandParser interface {
	ParseCommand(ctx context.Context, input string) Command
}

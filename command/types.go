package command

import "context"

type Command string

const (
	Cancel  Command = "cancel"
	Confirm Command = "confirm"
	None    Command = "none"
)

type Parser interface {
	ParseCommand(ctx context.Context, input string) (Command, error)
}

package command

import (
	"context"

	"github.com/abcxyz/pkg/cli"
)

// rootCmd defines the starting command structure.
var rootCmd = func() cli.Command {
	return &cli.RootCommand{
		Name:    "bufar",
		Version: "v0.0.0",
		Commands: map[string]cli.CommandFactory{
			"publish": func() cli.Command {
				return &PublishCommand{}
			},
			"generate": func() cli.Command {
				return &GenerateCommand{}
			},
		},
	}
}

// Run executes the CLI.
func Run(ctx context.Context, args []string) error {
	return rootCmd().Run(ctx, args) //nolint:wrapcheck // Want passthrough
}

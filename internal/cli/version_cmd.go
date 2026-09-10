package cli

import (
	"context"
	"fmt"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/output"
)

func versionCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "version", Usage: "show version information", Action: func(ctx context.Context, cmd *appcli.Command) error {
		if globalBool(cmd, "json") {
			r, err := output.New(deps.Out, true, false)
			if err != nil {
				return err
			}
			return r.JSON(map[string]string{"version": deps.Version, "commit": deps.Commit, "date": deps.BuildDate})
		}
		if globalBool(cmd, "plain") {
			_, err := fmt.Fprintln(deps.Out, deps.Version)
			return err
		}
		_, err := fmt.Fprintf(deps.Out, "youtrack version %s\ncommit: %s\nbuilt: %s\n", deps.Version, deps.Commit, deps.BuildDate)
		return err
	}}
}

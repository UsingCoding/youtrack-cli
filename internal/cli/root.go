package cli

import (
	"context"
	"time"

	appcli "github.com/urfave/cli/v3"
)

func NewRoot(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name:                  "youtrack",
		Usage:                 "YouTrack from your terminal — or your coding agent",
		Version:               deps.Version,
		EnableShellCompletion: true,
		Flags: []appcli.Flag{
			&appcli.StringFlag{Name: "profile", Usage: "YouTrack profile name"},
			&appcli.StringFlag{Name: "url", Usage: "override YouTrack service URL"},
			&appcli.StringFlag{Name: "token", Usage: "override YouTrack permanent token"},
			&appcli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "HTTP request timeout"},
			&appcli.BoolFlag{Name: "json", Usage: "emit stable JSON output"},
			&appcli.BoolFlag{Name: "plain", Usage: "emit shell-friendly plain output"},
			&appcli.BoolFlag{Name: "debug", Usage: "enable diagnostic errors"},
		},
		Commands: []*appcli.Command{
			authCommand(deps), issueCommand(deps), savedSearchCommand(deps), apiCommand(deps), configCommand(deps), skillCommand(deps), versionCommand(deps),
		},
		Action: func(ctx context.Context, cmd *appcli.Command) error {
			return appcli.ShowAppHelp(cmd)
		},
	}
}

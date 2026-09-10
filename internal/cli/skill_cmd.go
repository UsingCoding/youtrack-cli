package cli

import (
	"context"
	"fmt"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/output"
	"github.com/UsingCoding/youtrack-cli/internal/skill"
)

func skillCommand(deps Dependencies) *appcli.Command {
	projectFlag := func() appcli.Flag {
		return &appcli.BoolFlag{Name: "project", Usage: "operate on the current project instead of global agent directories"}
	}
	targetFlag := func() appcli.Flag {
		return &appcli.StringFlag{Name: "target", Usage: "agent target: claude, codex, cursor, opencode, agents"}
	}
	return &appcli.Command{Name: "skill", Usage: "manage the embedded coding-agent skill", Commands: []*appcli.Command{
		{Name: "list", Flags: []appcli.Flag{projectFlag()}, Action: func(ctx context.Context, cmd *appcli.Command) error {
			statuses, err := skill.List(cmd.Bool("project"))
			if err != nil {
				return err
			}
			if globalBool(cmd, "json") {
				r, err := output.New(deps.Out, true, false)
				if err != nil {
					return err
				}
				return r.JSON(statuses)
			}
			for _, s := range statuses {
				state := "not installed"
				if s.Exists {
					state = "installed"
				}
				if _, err := fmt.Fprintf(deps.Out, "%s\t%s\t%s\n", s.Target, state, s.Path); err != nil {
					return err
				}
			}
			return nil
		}},
		{Name: "install", Flags: []appcli.Flag{projectFlag(), targetFlag()}, Action: func(ctx context.Context, cmd *appcli.Command) error {
			paths, err := skill.Install(cmd.Bool("project"), cmd.String("target"))
			if err != nil {
				return err
			}
			for _, path := range paths {
				if _, err := fmt.Fprintf(deps.Out, "Installed youtrack-cli skill: %s\n", path); err != nil {
					return err
				}
			}
			return nil
		}},
		{Name: "update", Flags: []appcli.Flag{projectFlag(), targetFlag()}, Action: func(ctx context.Context, cmd *appcli.Command) error {
			paths, err := skill.Install(cmd.Bool("project"), cmd.String("target"))
			if err != nil {
				return err
			}
			for _, path := range paths {
				if _, err := fmt.Fprintf(deps.Out, "Updated youtrack-cli skill: %s\n", path); err != nil {
					return err
				}
			}
			return nil
		}},
		{Name: "remove", Flags: []appcli.Flag{projectFlag(), targetFlag()}, Action: func(ctx context.Context, cmd *appcli.Command) error {
			paths, err := skill.Remove(cmd.Bool("project"), cmd.String("target"))
			if err != nil {
				return err
			}
			for _, path := range paths {
				if _, err := fmt.Fprintf(deps.Out, "Removed youtrack-cli skill: %s\n", path); err != nil {
					return err
				}
			}
			return nil
		}},
	}}
}

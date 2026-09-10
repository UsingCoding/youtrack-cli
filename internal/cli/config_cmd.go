package cli

import (
	"context"
	"fmt"
	"sort"

	appcli "github.com/urfave/cli/v3"
)

func configCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "config", Usage: "inspect and edit local configuration", Commands: []*appcli.Command{
		{Name: "list", Action: func(ctx context.Context, cmd *appcli.Command) error {
			cfg, err := deps.Config.Load()
			if err != nil {
				return err
			}
			if globalBool(cmd, "json") {
				r, err := buildConfigRenderer(deps, cmd)
				if err != nil {
					return err
				}
				return r.JSON(cfg)
			}
			names := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				marker := " "
				if name == cfg.Current {
					marker = "*"
				}
				if _, err := fmt.Fprintf(deps.Out, "%s %s\t%s\n", marker, name, cfg.Profiles[name].URL); err != nil {
					return err
				}
			}
			return nil
		}},
		{Name: "get", ArgsUsage: "<key>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack config get <key>"); err != nil {
				return err
			}
			v, err := deps.Config.Get(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(deps.Out, v)
			return err
		}},
		{Name: "set", ArgsUsage: "<key> <value>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 2, 2, "youtrack config set <key> <value>"); err != nil {
				return err
			}
			return deps.Config.Set(args[0], args[1])
		}},
	}}
}

func buildConfigRenderer(deps Dependencies, cmd *appcli.Command) (*configRenderer, error) {
	return &configRenderer{out: deps.Out}, nil
}

type configRenderer struct {
	out interface{ Write([]byte) (int, error) }
}

func (r *configRenderer) JSON(v any) error {
	b, err := jsonMarshalIndent(v)
	if err != nil {
		return err
	}
	_, err = r.out.Write(append(b, '\n'))
	return err
}

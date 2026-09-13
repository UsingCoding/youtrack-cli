package cli

import (
	"context"
	"os"
	"strings"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func issueCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "issue", Usage: "inspect and edit issues", Commands: []*appcli.Command{
		issueViewCommand(deps), issueEditCommand(deps), issueMoveCommand(deps), issueFieldCommand(deps), issueTagCommand(deps),
	}}
}

func issueViewCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "view", Usage: "view an issue", ArgsUsage: "<issue>", Action: func(ctx context.Context, cmd *appcli.Command) error {
		args := cmd.Args().Slice()
		if err := requireArgs(args, 1, 1, "youtrack issue view <issue>"); err != nil {
			return err
		}
		rt, err := buildRuntime(deps, cmd)
		if err != nil {
			return err
		}
		issue, err := rt.service.GetIssue(ctx, domain.IssueRef(args[0]))
		if err != nil {
			return err
		}
		return rt.renderer.Issue(issue)
	}}
}

func issueEditCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name: "edit", Usage: "edit issue attributes, fields, and tags", ArgsUsage: "<issue>",
		DisableSliceFlagSeparator: true,
		Flags: []appcli.Flag{
			&appcli.StringFlag{Name: "summary"},
			&appcli.StringFlag{Name: "description"},
			&appcli.StringFlag{Name: "description-file"},
			&appcli.StringSliceFlag{Name: "field", Usage: "issue field assignment NAME=VALUE; repeat for multi-value fields"},
			&appcli.StringSliceFlag{Name: "tag", Usage: "tag to add"},
			&appcli.StringSliceFlag{Name: "remove-tag", Usage: "tag to remove"},
		},
		Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack issue edit <issue> [flags]"); err != nil {
				return err
			}
			if cmd.IsSet("description") && cmd.IsSet("description-file") {
				return app.Validationf("--description and --description-file are mutually exclusive")
			}
			var summary, description *string
			if cmd.IsSet("summary") {
				v := cmd.String("summary")
				summary = &v
			}
			if cmd.IsSet("description") {
				v := cmd.String("description")
				description = &v
			}
			if cmd.IsSet("description-file") {
				data, err := os.ReadFile(cmd.String("description-file"))
				if err != nil {
					return err
				}
				v := string(data)
				description = &v
			}
			inputs := make([]app.FieldInput, 0, len(cmd.StringSlice("field")))
			for _, raw := range cmd.StringSlice("field") {
				name, value, ok := strings.Cut(raw, "=")
				if !ok {
					return app.Validationf("invalid --field %q: expected NAME=VALUE", raw)
				}
				inputs = append(inputs, app.FieldInput{Name: name, Value: value})
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			issue, err := rt.service.EditIssue(ctx, domain.IssueRef(args[0]), app.EditRequest{
				Summary: summary, Description: description, Fields: inputs,
				AddTags: cmd.StringSlice("tag"), RemoveTags: cmd.StringSlice("remove-tag"),
			})
			if err != nil {
				return err
			}
			return rt.renderer.Issue(issue)
		},
	}
}

func issueMoveCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "move", Usage: "move an issue to another project", ArgsUsage: "<issue> <project>", Action: func(ctx context.Context, cmd *appcli.Command) error {
		args := cmd.Args().Slice()
		if err := requireArgs(args, 2, 2, "youtrack issue move <issue> <project>"); err != nil {
			return err
		}
		rt, err := buildRuntime(deps, cmd)
		if err != nil {
			return err
		}
		issue, err := rt.service.MoveIssue(ctx, domain.IssueRef(args[0]), domain.ProjectRef(args[1]))
		if err != nil {
			return err
		}
		return rt.renderer.Issue(issue)
	}}
}

func issueFieldCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "field", Usage: "inspect and edit issue fields", Commands: []*appcli.Command{
		{Name: "list", ArgsUsage: "<issue>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack issue field list <issue>"); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			fields, err := rt.service.ListFields(ctx, domain.IssueRef(args[0]))
			if err != nil {
				return err
			}
			return rt.renderer.Fields(fields)
		}},
		{Name: "get", ArgsUsage: "<issue> <field>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 2, 2, "youtrack issue field get <issue> <field>"); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			field, err := rt.service.GetField(ctx, domain.IssueRef(args[0]), domain.FieldRef(args[1]))
			if err != nil {
				return err
			}
			return rt.renderer.Field(field)
		}},
		{Name: "set", ArgsUsage: "<issue> <field> <value>...", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 3, -1, "youtrack issue field set <issue> <field> <value>..."); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			issue, err := rt.service.SetField(ctx, domain.IssueRef(args[0]), domain.FieldRef(args[1]), args[2:])
			if err != nil {
				return err
			}
			return rt.renderer.Issue(issue)
		}},
		{Name: "clear", ArgsUsage: "<issue> <field>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 2, 2, "youtrack issue field clear <issue> <field>"); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			issue, err := rt.service.ClearField(ctx, domain.IssueRef(args[0]), domain.FieldRef(args[1]))
			if err != nil {
				return err
			}
			return rt.renderer.Issue(issue)
		}},
	}}
}

func issueTagCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "tag", Usage: "manage issue tags", Commands: []*appcli.Command{
		{Name: "list", ArgsUsage: "<issue>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 1, 1, "youtrack issue tag list <issue>"); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			tags, err := rt.service.ListTags(ctx, domain.IssueRef(args[0]))
			if err != nil {
				return err
			}
			return rt.renderer.Tags(tags)
		}},
		{Name: "add", ArgsUsage: "<issue> <tag>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 2, 2, "youtrack issue tag add <issue> <tag>"); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			return rt.service.AddTag(ctx, domain.IssueRef(args[0]), args[1])
		}},
		{Name: "remove", ArgsUsage: "<issue> <tag>", Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			if err := requireArgs(args, 2, 2, "youtrack issue tag remove <issue> <tag>"); err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			return rt.service.RemoveTag(ctx, domain.IssueRef(args[0]), args[1])
		}},
	}}
}

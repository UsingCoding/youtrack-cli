package cli

import (
	"context"
	"os"
	"strings"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func issueTimeCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name:        "time",
		Usage:       "manage individual issue work items",
		Description: "List project work-item types, inspect logged time, and add, partially edit, or permanently remove individual work items.",
		Commands: []*appcli.Command{
			{
				Name:        "types",
				Usage:       "list project work-item types",
				Description: "Resolve the issue project and list its available work-item types.",
				ArgsUsage:   "<issue>",
				Flags:       paginationFlags(),
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 1, 1, "youtrack issue time types <issue> [--limit <n>] [--offset <n>] [--all]"); err != nil {
						return err
					}
					if strings.TrimSpace(args[0]) == "" {
						return app.Validationf("issue reference must not be blank")
					}
					request, err := pageRequest(cmd)
					if err != nil {
						return err
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					items, err := rt.service.ListWorkItemTypes(ctx, domain.IssueRef(args[0]), request)
					if err != nil {
						return err
					}
					return rt.renderer.WorkItemTypes(items)
				},
			},
			{
				Name:        "list",
				Usage:       "list work items logged on an issue",
				Description: "List individual work items; this does not expose or update a total custom field.",
				ArgsUsage:   "<issue>",
				Flags:       paginationFlags(),
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 1, 1, "youtrack issue time list <issue> [--limit <n>] [--offset <n>] [--all]"); err != nil {
						return err
					}
					if strings.TrimSpace(args[0]) == "" {
						return app.Validationf("issue reference must not be blank")
					}
					request, err := pageRequest(cmd)
					if err != nil {
						return err
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					items, err := rt.service.ListWorkItems(ctx, domain.IssueRef(args[0]), request)
					if err != nil {
						return err
					}
					return rt.renderer.WorkItems(items)
				},
			},
			{
				Name:        "view",
				Usage:       "view one work item by database ID",
				Description: "Read one work item under the supplied issue. Work-item IDs are database IDs, not list positions.",
				ArgsUsage:   "<issue> <work-item-id>",
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 2, 2, "youtrack issue time view <issue> <work-item-id>"); err != nil {
						return err
					}
					if err := validateWorkItemArgs(args); err != nil {
						return err
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					item, err := rt.service.GetWorkItem(ctx, domain.IssueRef(args[0]), args[1])
					if err != nil {
						return err
					}
					return rt.renderer.WorkItem(item)
				},
			},
			{
				Name:        "add",
				Usage:       "add a work item with an explicit date and duration",
				Description: "Log positive h/m duration on an explicit calendar date. The optional type is project-scoped; omitted author is chosen by the server.",
				ArgsUsage:   "<issue>",
				Flags: []appcli.Flag{
					&appcli.StringFlag{Name: "duration", Usage: "required positive h/m duration, for example 1h30m"},
					&appcli.StringFlag{Name: "date", Usage: "required calendar date in YYYY-MM-DD"},
					&appcli.StringFlag{Name: "text", Usage: "work-item text; may be empty; conflicts with --file"},
					&appcli.StringFlag{Name: "file", Usage: "read exact work-item text bytes from a file; conflicts with --text"},
					&appcli.StringFlag{Name: "type", Usage: "project work-item type database ID or name"},
					&appcli.StringFlag{Name: "author", Usage: "author database ID or login; use @me for the authenticated user"},
				},
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 1, 1, "youtrack issue time add <issue> --duration <period> --date <YYYY-MM-DD> [--text <text> | --file <path>] [--type <type>] [--author <user>]"); err != nil {
						return err
					}
					if strings.TrimSpace(args[0]) == "" {
						return app.Validationf("issue reference must not be blank")
					}
					if !cmd.IsSet("duration") || !cmd.IsSet("date") {
						return app.Validationf("--duration and --date are required")
					}
					text, err := workItemText(cmd)
					if err != nil {
						return err
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					item, err := rt.service.AddWorkItem(ctx, domain.IssueRef(args[0]), app.AddWorkItemRequest{
						Duration: cmd.String("duration"), Date: cmd.String("date"), Text: text, Type: workItemFlagString(cmd, "type"), Author: workItemFlagString(cmd, "author"),
					})
					if err != nil {
						return err
					}
					return rt.renderer.WorkItem(item)
				},
			},
			{
				Name:        "edit",
				Usage:       "partially update one work item",
				Description: "Replace only supplied values. Omitted values remain unchanged; --clear-type explicitly removes the current type.",
				ArgsUsage:   "<issue> <work-item-id>",
				Flags: []appcli.Flag{
					&appcli.StringFlag{Name: "duration", Usage: "replacement positive h/m duration, for example 1h30m"},
					&appcli.StringFlag{Name: "date", Usage: "replacement calendar date in YYYY-MM-DD"},
					&appcli.StringFlag{Name: "text", Usage: "replacement work-item text; may be empty; conflicts with --file"},
					&appcli.StringFlag{Name: "file", Usage: "read replacement text bytes from a file; conflicts with --text"},
					&appcli.StringFlag{Name: "type", Usage: "replacement project work-item type database ID or name; conflicts with --clear-type"},
					&appcli.BoolFlag{Name: "clear-type", Usage: "clear the current work-item type; conflicts with --type"},
					&appcli.StringFlag{Name: "author", Usage: "replacement author database ID or login; use @me for the authenticated user"},
				},
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 2, 2, "youtrack issue time edit <issue> <work-item-id> [--duration <period>] [--date <YYYY-MM-DD>] [--text <text> | --file <path>] [--type <type> | --clear-type] [--author <user>]"); err != nil {
						return err
					}
					if err := validateWorkItemArgs(args); err != nil {
						return err
					}
					if cmd.IsSet("type") && cmd.Bool("clear-type") {
						return app.Validationf("--type and --clear-type are mutually exclusive")
					}
					text, err := workItemText(cmd)
					if err != nil {
						return err
					}
					clearType := cmd.Bool("clear-type")
					if !cmd.IsSet("duration") && !cmd.IsSet("date") && text == nil && !cmd.IsSet("type") && !clearType && !cmd.IsSet("author") {
						return app.Validationf("at least one work item change is required")
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					item, err := rt.service.EditWorkItem(ctx, domain.IssueRef(args[0]), args[1], app.EditWorkItemRequest{
						Duration: workItemFlagString(cmd, "duration"), Date: workItemFlagString(cmd, "date"), Text: text, Type: workItemFlagString(cmd, "type"), ClearType: clearType, Author: workItemFlagString(cmd, "author"),
					})
					if err != nil {
						return err
					}
					return rt.renderer.WorkItem(item)
				},
			},
			{
				Name:        "remove",
				Usage:       "permanently delete one work item",
				Description: "Permanently delete a work item after reading it under the supplied issue. This operation cannot be restored.",
				ArgsUsage:   "<issue> <work-item-id>",
				Flags:       []appcli.Flag{&appcli.BoolFlag{Name: "yes", Usage: "required acknowledgement for permanent deletion"}},
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 2, 2, "youtrack issue time remove <issue> <work-item-id> --yes"); err != nil {
						return err
					}
					if err := validateWorkItemArgs(args); err != nil {
						return err
					}
					if !cmd.IsSet("yes") || !cmd.Bool("yes") {
						return app.Validationf("--yes is required for permanent work item removal")
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					id, err := rt.service.RemoveWorkItem(ctx, domain.IssueRef(args[0]), args[1])
					if err != nil {
						return err
					}
					return rt.renderer.WorkItemRemoval(id)
				},
			},
		},
	}
}

func validateWorkItemArgs(args []string) error {
	if strings.TrimSpace(args[0]) == "" {
		return app.Validationf("issue reference must not be blank")
	}
	if strings.TrimSpace(args[1]) == "" {
		return app.Validationf("work item ID must not be blank")
	}
	return nil
}

func workItemText(cmd *appcli.Command) (*string, error) {
	if cmd.IsSet("text") && cmd.IsSet("file") {
		return nil, app.Validationf("--text and --file are mutually exclusive")
	}
	if cmd.IsSet("text") {
		text := cmd.String("text")
		return &text, nil
	}
	if cmd.IsSet("file") {
		data, err := os.ReadFile(cmd.String("file"))
		if err != nil {
			return nil, err
		}
		text := string(data)
		return &text, nil
	}
	return nil, nil
}

func workItemFlagString(cmd *appcli.Command, name string) *string {
	if !cmd.IsSet(name) {
		return nil
	}
	value := cmd.String(name)
	return &value
}

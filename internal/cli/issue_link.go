package cli

import (
	"context"
	"strings"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func issueLinkCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{
		Name:        "link",
		Usage:       "discover and manage issue relationships",
		Description: "Discover configured link types and add, remove, or list one selected directed or undirected relationship.",
		Commands: []*appcli.Command{
			{
				Name: "types", Usage: "list configured issue link types", ArgsUsage: "[page flags]", Flags: paginationFlags(),
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					if err := requireArgs(cmd.Args().Slice(), 0, 0, "youtrack issue link types [--limit <n>] [--offset <n>] [--all]"); err != nil {
						return err
					}
					request, err := pageRequest(cmd)
					if err != nil {
						return err
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					items, err := rt.service.ListLinkTypes(ctx, request)
					if err != nil {
						return err
					}
					return rt.renderer.LinkTypes(items)
				},
			},
			{
				Name: "list", Usage: "list issues for one selected relationship", ArgsUsage: "<issue>", Flags: linkListFlags(),
				Action: func(ctx context.Context, cmd *appcli.Command) error {
					args := cmd.Args().Slice()
					if err := requireArgs(args, 1, 1, "youtrack issue link list <issue> --type <reference> [--direction outward|inward] [--limit <n>] [--offset <n>] [--all]"); err != nil {
						return err
					}
					if err := validateLinkIssueArgs(args); err != nil {
						return err
					}
					relation, err := linkRelationInput(cmd)
					if err != nil {
						return err
					}
					request, err := pageRequest(cmd)
					if err != nil {
						return err
					}
					rt, err := buildRuntime(deps, cmd)
					if err != nil {
						return err
					}
					items, err := rt.service.ListIssueLinks(ctx, domain.IssueRef(args[0]), relation, request)
					if err != nil {
						return err
					}
					return rt.renderer.IssueLinks(items)
				},
			},
			linkChangeCommand(deps, "add"),
			linkChangeCommand(deps, "remove"),
		},
	}
}

func linkChangeCommand(deps Dependencies, operation string) *appcli.Command {
	return &appcli.Command{
		Name: operation, Usage: operation + " one selected issue relationship", ArgsUsage: "<issue> <target-issue>", Flags: linkRelationFlags(),
		Action: func(ctx context.Context, cmd *appcli.Command) error {
			args := cmd.Args().Slice()
			usage := "youtrack issue link " + operation + " <issue> <target-issue> --type <reference> [--direction outward|inward]"
			if err := requireArgs(args, 2, 2, usage); err != nil {
				return err
			}
			if err := validateLinkIssueArgs(args); err != nil {
				return err
			}
			relation, err := linkRelationInput(cmd)
			if err != nil {
				return err
			}
			rt, err := buildRuntime(deps, cmd)
			if err != nil {
				return err
			}
			var change domain.IssueLinkChange
			if operation == "add" {
				change, err = rt.service.AddIssueLink(ctx, domain.IssueRef(args[0]), domain.IssueRef(args[1]), relation)
			} else {
				change, err = rt.service.RemoveIssueLink(ctx, domain.IssueRef(args[0]), domain.IssueRef(args[1]), relation)
			}
			if err != nil {
				return err
			}
			return rt.renderer.IssueLinkChange(change)
		},
	}
}

func linkListFlags() []appcli.Flag {
	flags := linkRelationFlags()
	return append(flags, paginationFlags()...)
}

func linkRelationFlags() []appcli.Flag {
	return []appcli.Flag{
		&appcli.StringFlag{Name: "type", Usage: "required link type database ID, name, or configured relation label"},
		&appcli.StringFlag{Name: "direction", Usage: "directed link direction: outward or inward"},
	}
}

func linkRelationInput(cmd *appcli.Command) (app.LinkRelationReference, error) {
	if !cmd.IsSet("type") || strings.TrimSpace(cmd.String("type")) == "" {
		return app.LinkRelationReference{}, app.Validationf("--type is required and must not be blank")
	}
	reference := app.LinkRelationReference{Type: cmd.String("type")}
	if !cmd.IsSet("direction") {
		return reference, nil
	}
	direction := cmd.String("direction")
	if direction != string(domain.LinkDirectionOutward) && direction != string(domain.LinkDirectionInward) {
		return app.LinkRelationReference{}, app.Validationf("--direction must be outward or inward")
	}
	reference.Direction = &direction
	return reference, nil
}

func validateLinkIssueArgs(args []string) error {
	if strings.TrimSpace(args[0]) == "" {
		return app.Validationf("issue reference must not be blank")
	}
	if len(args) > 1 && strings.TrimSpace(args[1]) == "" {
		return app.Validationf("target issue reference must not be blank")
	}
	return nil
}

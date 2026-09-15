package cli

import (
	"context"
	"os"
	"strings"

	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/domain"
)

func issueCommentCommand(deps Dependencies) *appcli.Command {
	return &appcli.Command{Name: "comment", Usage: "manage issue comments", Commands: []*appcli.Command{
		{
			Name: "list", ArgsUsage: "<issue>", Flags: paginationFlags(),
			Action: func(ctx context.Context, cmd *appcli.Command) error {
				args := cmd.Args().Slice()
				if err := requireArgs(args, 1, 1, "youtrack issue comment list <issue> [--limit <n>] [--offset <n>] [--all]"); err != nil {
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
				comments, err := rt.service.ListComments(ctx, domain.IssueRef(args[0]), request)
				if err != nil {
					return err
				}
				return rt.renderer.Comments(comments)
			},
		},
		{
			Name: "add", ArgsUsage: "<issue>", Flags: []appcli.Flag{&appcli.StringFlag{Name: "text"}, &appcli.StringFlag{Name: "file"}},
			Action: func(ctx context.Context, cmd *appcli.Command) error {
				args := cmd.Args().Slice()
				if err := requireArgs(args, 1, 1, "youtrack issue comment add <issue> (--text <text> | --file <path>)"); err != nil {
					return err
				}
				if strings.TrimSpace(args[0]) == "" {
					return app.Validationf("issue reference must not be blank")
				}
				textSet, fileSet := cmd.IsSet("text"), cmd.IsSet("file")
				if textSet == fileSet {
					return app.Validationf("exactly one of --text or --file is required")
				}
				text := cmd.String("text")
				if fileSet {
					data, err := os.ReadFile(cmd.String("file"))
					if err != nil {
						return err
					}
					text = string(data)
				}
				if strings.TrimSpace(text) == "" {
					return app.Validationf("comment text must not be blank")
				}
				rt, err := buildRuntime(deps, cmd)
				if err != nil {
					return err
				}
				comment, err := rt.service.AddComment(ctx, domain.IssueRef(args[0]), text)
				if err != nil {
					return err
				}
				return rt.renderer.Comment(comment)
			},
		},
		{
			Name: "remove", ArgsUsage: "<issue> <comment-entity-id>",
			Action: func(ctx context.Context, cmd *appcli.Command) error {
				args := cmd.Args().Slice()
				if err := requireArgs(args, 2, 2, "youtrack issue comment remove <issue> <comment-entity-id>"); err != nil {
					return err
				}
				if strings.TrimSpace(args[0]) == "" {
					return app.Validationf("issue reference must not be blank")
				}
				if strings.TrimSpace(args[1]) == "" {
					return app.Validationf("comment ID must not be blank")
				}
				rt, err := buildRuntime(deps, cmd)
				if err != nil {
					return err
				}
				if err := rt.service.RemoveComment(ctx, domain.IssueRef(args[0]), args[1]); err != nil {
					return err
				}
				return rt.renderer.CommentRemoval(args[1])
			},
		},
	}}
}

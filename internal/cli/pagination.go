package cli

import (
	appcli "github.com/urfave/cli/v3"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func paginationFlags() []appcli.Flag {
	return []appcli.Flag{
		&appcli.IntFlag{Name: "limit", Value: app.DefaultPageLimit},
		&appcli.IntFlag{Name: "offset", Value: 0},
		&appcli.BoolFlag{Name: "all"},
	}
}

func pageRequest(cmd *appcli.Command) (app.PageRequest, error) {
	request := app.PageRequest{Offset: cmd.Int("offset"), All: cmd.Bool("all")}
	if cmd.IsSet("limit") {
		request.Limit = new(cmd.Int("limit"))
	}
	if err := request.Validate(); err != nil {
		return app.PageRequest{}, err
	}
	return request, nil
}

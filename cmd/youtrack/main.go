package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/app"
	cliapp "github.com/UsingCoding/youtrack-cli/internal/cli"
	"github.com/UsingCoding/youtrack-cli/internal/config"
	"github.com/UsingCoding/youtrack-cli/internal/credentials"
	"github.com/UsingCoding/youtrack-cli/internal/version"
	"github.com/UsingCoding/youtrack-cli/internal/youtrack"
)

func main() {
	configStore, err := config.NewStore("")
	if err != nil {
		fail(err, debugRequested(os.Args))
	}
	credentialStore, err := credentials.NewStore("")
	if err != nil {
		fail(err, debugRequested(os.Args))
	}
	root := cliapp.NewRoot(cliapp.Dependencies{
		Config: configStore, Credentials: credentialStore,
		In: os.Stdin, Out: os.Stdout, Err: os.Stderr,
		Version: version.Version, Commit: version.Commit, BuildDate: version.Date,
	})
	if err := root.Run(context.Background(), os.Args); err != nil {
		fail(err, debugRequested(os.Args))
	}
}

func fail(err error, debug bool) {
	fmt.Fprintln(os.Stderr, "error:", err)
	if debug {
		var apiErr *youtrack.APIError
		if errors.As(err, &apiErr) {
			fmt.Fprintln(os.Stderr, apiErr.DebugString())
		}
	}
	code := 1
	switch app.KindOf(err) {
	case app.ErrorValidation:
		code = 2
	case app.ErrorAuth:
		code = 3
	case app.ErrorNotFound:
		code = 4
	case app.ErrorAmbiguous:
		code = 5
	}
	os.Exit(code)
}

func debugRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--debug" || strings.EqualFold(arg, "--debug=true") {
			return true
		}
	}
	return false
}

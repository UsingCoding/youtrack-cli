package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	appcli "github.com/urfave/cli/v3"

	"golang.org/x/term"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

func requireArgs(args []string, min, max int, usage string) error {
	if len(args) < min || (max >= 0 && len(args) > max) {
		return app.Validationf("usage: %s", usage)
	}
	return nil
}

func readLine(in io.Reader, out io.Writer, prompt string) (string, error) {
	if prompt != "" {
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return "", err
		}
	}
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func readSecret(in io.Reader, out io.Writer, prompt string) (string, error) {
	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if _, err := fmt.Fprint(out, prompt); err != nil {
			return "", err
		}
		data, err := term.ReadPassword(int(file.Fd()))
		if _, err := fmt.Fprintln(out); err != nil {
			return "", err
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	return readLine(in, out, prompt)
}

func globalCommand(cmd *appcli.Command) *appcli.Command {
	lineage := cmd.Lineage()
	if len(lineage) == 0 {
		return cmd
	}
	return lineage[len(lineage)-1]
}

func globalString(cmd *appcli.Command, name string) string {
	for _, current := range cmd.Lineage() {
		if current.IsSet(name) {
			return current.String(name)
		}
	}
	return globalCommand(cmd).String(name)
}

func globalBool(cmd *appcli.Command, name string) bool {
	for _, current := range cmd.Lineage() {
		if current.IsSet(name) {
			return current.Bool(name)
		}
	}
	return globalCommand(cmd).Bool(name)
}

func globalDuration(cmd *appcli.Command, name string) time.Duration {
	for _, current := range cmd.Lineage() {
		if current.IsSet(name) {
			return current.Duration(name)
		}
	}
	return globalCommand(cmd).Duration(name)
}

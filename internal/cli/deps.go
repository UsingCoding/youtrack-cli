package cli

import (
	"io"
	"net/http"

	"github.com/UsingCoding/youtrack-cli/internal/config"
)

type ConfigStore interface {
	Load() (config.Config, error)
	Save(config.Config) error
	Get(string) (string, error)
	Set(string, string) error
}

type CredentialStore interface {
	Get(string) (string, error)
	Set(string, string) error
	Delete(string) error
}

type Dependencies struct {
	Config      ConfigStore
	Credentials CredentialStore
	In          io.Reader
	Out         io.Writer
	Err         io.Writer
	HTTPClient  *http.Client
	Version     string
	Commit      string
	BuildDate   string
}

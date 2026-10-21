package cli

import (
	"fmt"
	"io"
	"net/http"
	goruntime "runtime"

	"github.com/UsingCoding/youtrack-cli/internal/browser"
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
	Config        ConfigStore
	Credentials   CredentialStore
	BrowserOpener browser.Opener
	In            io.Reader
	Out           io.Writer
	Err           io.Writer
	HTTPClient    *http.Client
	Version       string
	Commit        string
	BuildDate     string
}

// userAgent is sent to YouTrack. The leading "youtrack-cli/<version>" token is parsed by youtrack-proxy, keep it stable.
func (d Dependencies) userAgent() string {
	v, commit := d.Version, d.Commit
	if v == "" {
		v = "dev"
	}
	if commit == "" {
		commit = "none"
	}
	return fmt.Sprintf("youtrack-cli/%s (%s; %s/%s)", v, commit, goruntime.GOOS, goruntime.GOARCH)
}

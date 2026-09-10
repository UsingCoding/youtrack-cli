package config

import (
	"fmt"
	"os"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

type CredentialReader interface {
	Get(profile string) (string, error)
}

type RuntimeInput struct {
	Profile string
	URL     string
	Token   string
}

type Runtime struct {
	Profile string
	URL     string
	Token   string
}

func Resolve(cfg Config, creds CredentialReader, in RuntimeInput) (Runtime, error) {
	profile := in.Profile
	if profile == "" {
		profile = os.Getenv("YOUTRACK_PROFILE")
	}
	if profile == "" {
		profile = cfg.Current
	}

	explicitURL := in.URL
	if explicitURL == "" {
		explicitURL = os.Getenv("YOUTRACK_URL")
	}
	explicitToken := in.Token
	if explicitToken == "" {
		explicitToken = os.Getenv("YOUTRACK_TOKEN")
	}

	if explicitURL != "" {
		if explicitToken == "" {
			return Runtime{}, app.Authf("YOUTRACK_URL/--url requires YOUTRACK_TOKEN/--token; stored credentials are not reused for an overridden server")
		}
		return Runtime{Profile: profile, URL: explicitURL, Token: explicitToken}, nil
	}

	if profile == "" {
		return Runtime{}, app.Authf("no YouTrack profile is selected; run 'youtrack auth login'")
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return Runtime{}, app.Authf("YouTrack profile %q does not exist", profile)
	}
	if p.URL == "" {
		return Runtime{}, app.Authf("YouTrack profile %q has no URL", profile)
	}
	if explicitToken != "" {
		return Runtime{Profile: profile, URL: p.URL, Token: explicitToken}, nil
	}
	token, err := creds.Get(profile)
	if err != nil {
		return Runtime{}, fmt.Errorf("read credential for profile %q: %w", profile, err)
	}
	if token == "" {
		return Runtime{}, app.Authf("no token is stored for profile %q; run 'youtrack auth login'", profile)
	}
	return Runtime{Profile: profile, URL: p.URL, Token: token}, nil
}

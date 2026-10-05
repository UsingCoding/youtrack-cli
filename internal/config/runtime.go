package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/UsingCoding/youtrack-cli/internal/app"
)

type CredentialReader interface {
	Get(profile string) (string, error)
}

type ServiceInput struct {
	Profile string
	URL     string
}

type Service struct {
	Profile string
	URL     string
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

func ResolveService(cfg Config, in ServiceInput) (Service, error) {
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
	if explicitURL != "" {
		return Service{Profile: profile, URL: explicitURL}, nil
	}

	if profile == "" {
		return Service{}, app.Authf("no YouTrack profile is selected; run 'youtrack auth login'")
	}
	p, ok := cfg.Profiles[profile]
	if !ok {
		return Service{}, app.Authf("YouTrack profile %q does not exist", profile)
	}
	if strings.TrimSpace(p.URL) == "" {
		return Service{}, app.Authf("YouTrack profile %q has no URL", profile)
	}
	return Service{Profile: profile, URL: p.URL}, nil
}

func Resolve(cfg Config, creds CredentialReader, in RuntimeInput) (Runtime, error) {
	service, err := ResolveService(cfg, ServiceInput{Profile: in.Profile, URL: in.URL})
	if err != nil {
		return Runtime{}, err
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
		return Runtime{Profile: service.Profile, URL: service.URL, Token: explicitToken}, nil
	}

	if explicitToken != "" {
		return Runtime{Profile: service.Profile, URL: service.URL, Token: explicitToken}, nil
	}
	token, err := creds.Get(service.Profile)
	if err != nil {
		return Runtime{}, fmt.Errorf("read credential for profile %q: %w", service.Profile, err)
	}
	if token == "" {
		return Runtime{}, app.Authf("no token is stored for profile %q; run 'youtrack auth login'", service.Profile)
	}
	return Runtime{Profile: service.Profile, URL: service.URL, Token: token}, nil
}

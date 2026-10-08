package cli

import (
	"github.com/UsingCoding/youtrack-cli/internal/app"
	"github.com/UsingCoding/youtrack-cli/internal/output"
)

func openURL(deps Dependencies, renderer *output.Renderer, url string, printURL bool) error {
	if printURL {
		return renderer.Open(url, false)
	}
	if err := deps.BrowserOpener.Open(url); err != nil {
		return app.Runtimef("dispatch browser opener: %v; use --print-url to print the destination instead", err)
	}
	return renderer.Open(url, true)
}

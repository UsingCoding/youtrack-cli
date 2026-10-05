package output

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenRendererJSONAndPlainContracts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		json   bool
		plain  bool
		opened bool
		want   string
	}{
		{name: "JSON dispatched", json: true, opened: true, want: "{\n  \"url\": \"https://youtrack.example/issues?q=project%3A+APP\",\n  \"opened\": true\n}\n"},
		{name: "JSON print only", json: true, opened: false, want: "{\n  \"url\": \"https://youtrack.example/issues?q=project%3A+APP\",\n  \"opened\": false\n}\n"},
		{name: "plain", plain: true, opened: true, want: "https://youtrack.example/issues?q=project%3A+APP\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			renderer, err := New(&out, tc.json, tc.plain)
			require.NoError(t, err)

			require.NoError(t, renderer.Open("https://youtrack.example/issues?q=project%3A+APP", tc.opened))

			assert.Equal(t, tc.want, out.String())
		})
	}
}

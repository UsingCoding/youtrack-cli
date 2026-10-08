package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildIssueURLPreservesOriginAndContext(t *testing.T) {
	for _, tc := range []struct {
		name string
		base string
		id   string
		want string
	}{
		{name: "root", base: "https://youtrack.example", id: "APP-1", want: "https://youtrack.example/issue/APP-1"},
		{name: "context path", base: "https://youtrack.example/youtrack", id: "APP-1", want: "https://youtrack.example/youtrack/issue/APP-1"},
		{name: "escaped segment", base: "https://youtrack.example/youtrack/", id: "APP/1%#", want: "https://youtrack.example/youtrack/issue/APP%2F1%25%23"},
		{name: "dot segment", base: "https://youtrack.example/youtrack", id: "..", want: "https://youtrack.example/youtrack/issue/%2E%2E"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildIssueURL(tc.base, tc.id)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildSearchURLEncodesOriginalQueryOnce(t *testing.T) {
	got, err := BuildSearchURL("https://youtrack.example/youtrack", "project: APP #Open&+%{}\"✓\n")
	require.NoError(t, err)
	assert.Equal(t, "https://youtrack.example/youtrack/issues?q=project%3A+APP+%23Open%26%2B%25%7B%7D%22%E2%9C%93%0A", got)
}

func TestBuildSearchURLPreservesRootAndContext(t *testing.T) {
	for _, tc := range []struct {
		base string
		want string
	}{
		{base: "https://youtrack.example", want: "https://youtrack.example/issues?q=project%3A+APP"},
		{base: "https://youtrack.example/youtrack/", want: "https://youtrack.example/youtrack/issues?q=project%3A+APP"},
	} {
		got, err := BuildSearchURL(tc.base, "project: APP")
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}

func TestBuildersRejectUnsafeInput(t *testing.T) {
	for _, base := range []string{
		"", "youtrack.example", "ftp://youtrack.example", "https:///youtrack", "https://user:secret@youtrack.example", "https://youtrack.example?x=1", "https://youtrack.example#fragment",
	} {
		t.Run(base, func(t *testing.T) {
			_, err := BuildIssueURL(base, "APP-1")
			require.Error(t, err)
			_, err = BuildSearchURL(base, "project: APP")
			require.Error(t, err)
		})
	}
	for _, value := range []string{"", " \t\n "} {
		t.Run("blank issue "+value, func(t *testing.T) {
			_, err := BuildIssueURL("https://youtrack.example", value)
			require.Error(t, err)
		})
		t.Run("blank query "+value, func(t *testing.T) {
			_, err := BuildSearchURL("https://youtrack.example", value)
			require.Error(t, err)
		})
	}
}

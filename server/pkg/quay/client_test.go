package quay

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient spins up an httptest server and returns
// a Client pointed at it, plus a cleanup function.
func newTestClient(
	t *testing.T,
	handler http.HandlerFunc,
) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := NewClient(
		slog.Default(),
	).WithBaseURL(srv.URL)

	return c, srv
}

func TestListRepos(t *testing.T) {
	// Verify that only arcaflow-plugin-* repos are
	// returned and that excluded repos are filtered.
	body := `{
		"repositories": [
			{
				"namespace": "arcalot",
				"name": "arcaflow-plugin-stressng",
				"description": "stress plugin"
			},
			{
				"namespace": "arcalot",
				"name": "arcaflow-plugin-baseimage-python-buildbase",
				"description": "base image"
			},
			{
				"namespace": "arcalot",
				"name": "arcaflow-engine",
				"description": "engine"
			},
			{
				"namespace": "arcalot",
				"name": "arcaflow-plugin-template-python",
				"description": "template"
			}
		]
	}`

	c, _ := newTestClient(t, func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set(
			"Content-Type", "application/json",
		)
		_, _ = w.Write([]byte(body))
	})

	repos, err := c.ListRepos(
		context.Background(), "arcalot",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf(
			"expected 1 repo, got %d", len(repos),
		)
	}
	if repos[0].Name != "arcaflow-plugin-stressng" {
		t.Errorf(
			"unexpected repo name: %s", repos[0].Name,
		)
	}
}

func TestListReposExcludesNonPlugins(t *testing.T) {
	// Repos without the arcaflow-plugin- prefix must
	// never appear in the result set.
	body := `{
		"repositories": [
			{
				"namespace": "arcalot",
				"name": "some-other-tool",
				"description": "not a plugin"
			},
			{
				"namespace": "arcalot",
				"name": "arcaflow-engine",
				"description": "engine"
			}
		]
	}`

	c, _ := newTestClient(t, func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set(
			"Content-Type", "application/json",
		)
		_, _ = w.Write([]byte(body))
	})

	repos, err := c.ListRepos(
		context.Background(), "arcalot",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 0 {
		t.Fatalf(
			"expected 0 repos, got %d", len(repos),
		)
	}
}

func TestListReposHTTPError(t *testing.T) {
	c, _ := newTestClient(t, func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	_, err := c.ListRepos(
		context.Background(), "arcalot",
	)
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestListTags(t *testing.T) {
	body := `{
		"tags": [
			{
				"name": "0.9.0",
				"manifest_digest": "sha256:abc"
			},
			{
				"name": "latest",
				"manifest_digest": "sha256:def"
			}
		]
	}`

	c, _ := newTestClient(t, func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.Header().Set(
			"Content-Type", "application/json",
		)
		_, _ = w.Write([]byte(body))
	})

	tags, err := c.ListTags(
		context.Background(),
		"arcalot", "arcaflow-plugin-stressng",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf(
			"expected 2 tags, got %d", len(tags),
		)
	}
	if tags[0].Name != "0.9.0" {
		t.Errorf(
			"unexpected tag name: %s", tags[0].Name,
		)
	}
}

func TestLatestSemver(t *testing.T) {
	tests := []struct {
		name string
		tags []Tag
		want string
	}{
		{
			name: "picks highest version",
			tags: []Tag{
				{Name: "0.8.0"},
				{Name: "0.9.0"},
				{Name: "0.9.1"},
				{Name: "latest"},
				{Name: "main_latest"},
				{Name: "feat_branch_build"},
			},
			want: "0.9.1",
		},
		{
			name: "major version wins",
			tags: []Tag{
				{Name: "1.0.0"},
				{Name: "0.99.99"},
			},
			want: "1.0.0",
		},
		{
			name: "minor version tiebreak",
			tags: []Tag{
				{Name: "1.1.0"},
				{Name: "1.2.0"},
			},
			want: "1.2.0",
		},
	}

	c := NewClient(slog.Default())
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := c.LatestSemver(tc.tags)
			if got != tc.want {
				t.Errorf(
					"got %q, want %q", got, tc.want,
				)
			}
		})
	}
}

func TestLatestSemverNoValidTags(t *testing.T) {
	c := NewClient(slog.Default())
	tags := []Tag{
		{Name: "latest"},
		{Name: "main_latest"},
		{Name: "some_branch"},
		{Name: "not-semver"},
	}
	got := c.LatestSemver(tags)
	if got != "" {
		t.Errorf(
			"expected empty string, got %q", got,
		)
	}
}

func TestLatestSemverWithVPrefix(t *testing.T) {
	c := NewClient(slog.Default())
	tags := []Tag{
		{Name: "v0.9.0"},
		{Name: "v0.8.0"},
		{Name: "0.7.0"},
	}
	got := c.LatestSemver(tags)
	if got != "0.9.0" {
		t.Errorf(
			"expected 0.9.0, got %q", got,
		)
	}
}

// Package quay provides a client for the Quay.io REST API,
// focused on discovering Arcaflow plugin repositories and
// their available tags.
package quay

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// excludedRepos lists repositories that match the plugin
// prefix but are not actual runnable plugins (base images,
// templates, test scaffolds).
var excludedRepos = map[string]bool{
	"arcaflow-plugin-baseimage-python-buildbase": true,
	"arcaflow-plugin-baseimage-python-osbase":    true,
	"arcaflow-plugin-template-python":            true,
	"arcaflow-plugin-test-impl-go":               true,
}

// Repository represents a Quay.io repository.
type Repository struct {
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Tag represents a container image tag.
type Tag struct {
	Name           string `json:"name"`
	ManifestDigest string `json:"manifest_digest"`
}

// listReposResponse is the JSON envelope returned by
// the Quay repository list endpoint.
type listReposResponse struct {
	Repositories []Repository `json:"repositories"`
}

// listTagsResponse is the JSON envelope returned by
// the Quay tag list endpoint.
type listTagsResponse struct {
	Tags []Tag `json:"tags"`
}

// Client queries the Quay.io REST API for plugin
// repositories. All HTTP calls are serialized via a
// mutex to stay within Quay rate limits.
type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger
	// serialize Quay requests to avoid rate limits
	mu sync.Mutex
}

// NewClient returns a Client configured for the public
// Quay.io API with a 15-second HTTP timeout.
func NewClient(logger *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "https://quay.io",
		logger:  logger,
	}
}

// WithBaseURL overrides the default Quay base URL.
// Useful for pointing at a test server.
func (c *Client) WithBaseURL(url string) *Client {
	c.baseURL = url
	return c
}

// WithHTTPClient overrides the default HTTP client.
func (c *Client) WithHTTPClient(
	hc *http.Client,
) *Client {
	c.httpClient = hc
	return c
}

// ListRepos returns all public Arcaflow plugin
// repositories in the given Quay organisation. Repos
// that are base images, templates, or test scaffolds
// are excluded.
func (c *Client) ListRepos(
	ctx context.Context,
	org string,
) ([]Repository, error) {
	url := fmt.Sprintf(
		"%s/api/v1/repository?namespace=%s&public=true",
		c.baseURL, org,
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.DebugContext(ctx,
		"listing repos", "org", org,
	)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, url, nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"quay: build request: %w", err,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"quay: list repos: %w", err,
		)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"quay: list repos: HTTP %d", resp.StatusCode,
		)
	}

	var body listReposResponse
	if err := json.NewDecoder(
		resp.Body,
	).Decode(&body); err != nil {
		return nil, fmt.Errorf(
			"quay: decode repos: %w", err,
		)
	}

	filtered := make([]Repository, 0, len(body.Repositories))
	for _, r := range body.Repositories {
		if !strings.HasPrefix(
			r.Name, "arcaflow-plugin-",
		) {
			continue
		}
		if excludedRepos[r.Name] {
			continue
		}
		filtered = append(filtered, r)
	}

	c.logger.DebugContext(ctx,
		"repos found",
		"total", len(body.Repositories),
		"filtered", len(filtered),
	)

	return filtered, nil
}

// ListTags returns all active tags for a repository.
func (c *Client) ListTags(
	ctx context.Context,
	org, repo string,
) ([]Tag, error) {
	url := fmt.Sprintf(
		"%s/api/v1/repository/%s/%s/tag/"+
			"?onlyActiveTags=true&limit=100",
		c.baseURL, org, repo,
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.DebugContext(ctx,
		"listing tags", "org", org, "repo", repo,
	)

	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, url, nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"quay: build request: %w", err,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"quay: list tags: %w", err,
		)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"quay: list tags: HTTP %d", resp.StatusCode,
		)
	}

	var body listTagsResponse
	if err := json.NewDecoder(
		resp.Body,
	).Decode(&body); err != nil {
		return nil, fmt.Errorf(
			"quay: decode tags: %w", err,
		)
	}

	return body.Tags, nil
}

// semver holds a parsed major.minor.patch triple.
type semver struct {
	Major, Minor, Patch int
}

// parseSemver extracts a semver triple from a string
// such as "1.2.3" or "v1.2.3". It returns false when
// the string is not a valid three-part version.
func parseSemver(s string) (semver, bool) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return semver{}, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, false
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semver{}, false
	}
	return semver{major, minor, patch}, true
}

// less reports whether a is strictly less than b.
func (a semver) less(b semver) bool {
	if a.Major != b.Major {
		return a.Major < b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor < b.Minor
	}
	return a.Patch < b.Patch
}

// LatestSemver finds the highest semantic version among
// the supplied tags. Tags named "latest", "main_latest",
// or containing underscores (branch builds) are skipped.
// Returns the version string without a "v" prefix, or
// an empty string if no valid semver tag is found.
func (c *Client) LatestSemver(tags []Tag) string {
	var best semver
	found := false

	for _, t := range tags {
		if t.Name == "latest" ||
			t.Name == "main_latest" {
			continue
		}
		if strings.Contains(t.Name, "_") {
			continue
		}

		v, ok := parseSemver(t.Name)
		if !ok {
			continue
		}
		if !found || best.less(v) {
			best = v
			found = true
		}
	}

	if !found {
		return ""
	}

	return fmt.Sprintf(
		"%d.%d.%d", best.Major, best.Minor, best.Patch,
	)
}

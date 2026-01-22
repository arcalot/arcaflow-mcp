package workflow

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"log/slog"
)

func TestLoadFromFilesystem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(root, "a.yaml"),
		[]byte("name: workflow-a"),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "nested", "b.yml"),
		[]byte("name: workflow-b"),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "nested", "c.json"),
		[]byte(`{"name":"workflow-c"}`),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "ignored.txt"),
		[]byte("skip"),
		0o644,
	); err != nil {
		t.Fatalf("write ignored: %v", err)
	}

	loader := NewLoader()
	index, err := loader.LoadFromFilesystem(ctx, root)
	if err != nil {
		t.Fatalf("load workflows: %v", err)
	}

	if got := len(index.Workflows); got != 3 {
		t.Fatalf("expected 3 workflows, got %d", got)
	}
	if index.Workflows[0].Path != "a.yaml" {
		t.Fatalf("expected first workflow path a.yaml, got %s", index.Workflows[0].Path)
	}
	if index.Workflows[1].Path != filepath.Join("nested", "b.yml") {
		t.Fatalf("expected second workflow path nested/b.yml, got %s", index.Workflows[1].Path)
	}
	if index.Workflows[2].Path != filepath.Join("nested", "c.json") {
		t.Fatalf("expected third workflow path nested/c.json, got %s", index.Workflows[2].Path)
	}
	for _, workflow := range index.Workflows {
		if workflow.Source.Kind != SourceFilesystem {
			t.Fatalf("expected filesystem source kind, got %s", workflow.Source.Kind)
		}
		if workflow.ContentSHA256 == "" {
			t.Fatalf("expected workflow content hash")
		}
	}
}

func TestLoadFromURLUsesETagCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	requests := 0
	var observedIfNoneMatch string
	etag := `"v1"`
	lastModified := time.Date(2026, 1, 22, 12, 0, 0, 0, time.UTC)

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests++
			if requests > 1 {
				observedIfNoneMatch = req.Header.Get("If-None-Match")
				if observedIfNoneMatch == etag {
					return &http.Response{
						StatusCode: http.StatusNotModified,
						Header:     make(http.Header),
						Body:       io.NopCloser(bytes.NewBuffer(nil)),
						Request:    req,
					}, nil
				}
			}

			headers := make(http.Header)
			headers.Set("ETag", etag)
			headers.Set("Last-Modified", lastModified.Format(http.TimeFormat))
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     headers,
				Body:       io.NopCloser(bytes.NewBufferString("workflow: url")),
				Request:    req,
			}, nil
		}),
	}

	loader := NewLoader(WithHTTPClient(client))
	first, err := loader.LoadFromURL(ctx, "https://example.com/workflow.yaml")
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}

	second, err := loader.LoadFromURL(ctx, "https://example.com/workflow.yaml")
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}

	if observedIfNoneMatch != etag {
		t.Fatalf("expected If-None-Match %s, got %s", etag, observedIfNoneMatch)
	}
	if len(first.Workflows) != 1 || len(second.Workflows) != 1 {
		t.Fatalf("expected 1 workflow in each response")
	}
	if first.Workflows[0].ContentSHA256 != second.Workflows[0].ContentSHA256 {
		t.Fatalf("expected cached workflow content hash to match")
	}
}

func TestLoadFromGit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	cacheDir := t.TempDir()

	fake := &fakeGitClient{
		commit: "abc123",
		files: map[string][]byte{
			filepath.Join("workflows", "sample.yaml"): []byte("name: git"),
			filepath.Join("workflows", "sample.json"): []byte(`{"name":"git-json"}`),
			"README.md": []byte("ignore"),
		},
	}

	loader := NewLoader(
		WithGitClient(fake),
		WithGitCacheDir(cacheDir),
	)

	index, err := loader.LoadFromGit(
		ctx,
		"https://example.com/repo.git",
		"main",
		"workflows",
	)
	if err != nil {
		t.Fatalf("load workflows: %v", err)
	}

	if got := len(index.Workflows); got != 2 {
		t.Fatalf("expected 2 workflows, got %d", got)
	}
	if index.Source.Kind != SourceGit {
		t.Fatalf("expected git source kind, got %s", index.Source.Kind)
	}
	if index.Snapshot.Commit != fake.commit {
		t.Fatalf("expected commit %s, got %s", fake.commit, index.Snapshot.Commit)
	}
}

func TestLoadFromFilesystemSingleFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "single.yaml")
	if err := os.WriteFile(path, []byte("name: single"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}

	now := time.Date(2026, 1, 22, 12, 0, 0, 0, time.UTC)
	loader := NewLoader(WithNow(func() time.Time { return now }))
	index, err := loader.LoadFromFilesystem(ctx, path)
	if err != nil {
		t.Fatalf("load workflow: %v", err)
	}
	if len(index.Workflows) != 1 {
		t.Fatalf("expected 1 workflow, got %d", len(index.Workflows))
	}
	if index.FetchedAt != now.UTC() {
		t.Fatalf("expected fetched time to use now override")
	}
}

func TestLoadFromFilesystemRejectsNonWorkflowFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	loader := NewLoader()
	if _, err := loader.LoadFromFilesystem(ctx, path); err == nil {
		t.Fatalf("expected error for non-workflow file")
	}
}

func TestLoaderOptions(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cache := NewMemoryCache()
	now := func() time.Time { return time.Unix(0, 0) }

	loader := NewLoader(
		WithLogger(logger),
		WithCache(cache),
		WithNow(now),
	)
	if loader.logger != logger {
		t.Fatalf("expected logger override")
	}
	if loader.cache != cache {
		t.Fatalf("expected cache override")
	}
	if loader.now() != now() {
		t.Fatalf("expected now override")
	}
}

func TestLoadDirectoryUsesFingerprint(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(root, "workflow.yaml"),
		[]byte("name: workflow"),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	loader := NewLoader()
	index, err := loader.loadDirectory(
		ctx,
		"cache-key",
		root,
		root,
		SourceMetadata{Kind: SourceFilesystem, Location: root},
	)
	if err != nil {
		t.Fatalf("load directory: %v", err)
	}
	if index.Snapshot.FilesystemFingerprint == "" {
		t.Fatalf("expected filesystem fingerprint")
	}
}

func TestParseHTTPTime(t *testing.T) {
	t.Parallel()
	if parsed := parseHTTPTime("not-a-time"); !parsed.IsZero() {
		t.Fatalf("expected zero time for invalid value")
	}
	now := time.Date(2026, 1, 22, 12, 0, 0, 0, time.UTC)
	value := now.Format(http.TimeFormat)
	if parsed := parseHTTPTime(value); parsed.IsZero() {
		t.Fatalf("expected parsed time")
	}
}

func TestLoadDirectoryCacheHit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(root, "workflow.yaml"),
		[]byte("name: workflow"),
		0o644,
	); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	now := time.Date(2026, 1, 22, 12, 0, 0, 0, time.UTC)
	loader := NewLoader(WithNow(func() time.Time { return now }))
	source := SourceMetadata{Kind: SourceFilesystem, Location: root}

	first, err := loader.loadDirectory(ctx, "cache-key", root, root, source)
	if err != nil {
		t.Fatalf("load directory: %v", err)
	}
	second, err := loader.loadDirectory(ctx, "cache-key", root, root, source)
	if err != nil {
		t.Fatalf("load directory: %v", err)
	}
	if first.FetchedAt != second.FetchedAt {
		t.Fatalf("expected cached fetch time")
	}
}

type fakeGitClient struct {
	files  map[string][]byte
	commit string
}

func (client *fakeGitClient) Sync(
	ctx context.Context,
	repoURL string,
	destDir string,
	ref string,
) (string, error) {
	_ = ctx
	_ = repoURL
	_ = ref
	for path, content := range client.files {
		fullPath := filepath.Join(destDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, content, 0o644); err != nil {
			return "", err
		}
	}
	return client.commit, nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return roundTripper(request)
}

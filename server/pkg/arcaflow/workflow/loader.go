package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout = 30 * time.Second
)

// Loader discovers workflows from multiple sources.
type Loader struct {
	logger     *slog.Logger
	cache      Cache
	httpClient *http.Client
	gitClient  GitClient
	now        func() time.Time
	gitCache   string
}

// LoaderOption configures workflow loader behavior.
type LoaderOption func(*Loader)

// NewLoader constructs a workflow loader with sensible defaults.
func NewLoader(options ...LoaderOption) *Loader {
	loader := &Loader{
		logger: slog.Default(),
		cache:  NewMemoryCache(),
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		gitClient: NewGitClient(),
		now:       time.Now,
		gitCache:  filepath.Join(os.TempDir(), "arcaflow-mcp-workflows"),
	}
	for _, option := range options {
		if option != nil {
			option(loader)
		}
	}
	return loader
}

// WithLogger sets the loader logger.
func WithLogger(logger *slog.Logger) LoaderOption {
	return func(loader *Loader) {
		if logger != nil {
			loader.logger = logger
		}
	}
}

// WithCache sets the loader cache implementation.
func WithCache(cache Cache) LoaderOption {
	return func(loader *Loader) {
		if cache != nil {
			loader.cache = cache
		}
	}
}

// WithHTTPClient sets the HTTP client used for URL sources.
func WithHTTPClient(client *http.Client) LoaderOption {
	return func(loader *Loader) {
		if client != nil {
			loader.httpClient = client
		}
	}
}

// WithGitClient sets the git client used for git sources.
func WithGitClient(client GitClient) LoaderOption {
	return func(loader *Loader) {
		if client != nil {
			loader.gitClient = client
		}
	}
}

// WithNow overrides the time provider for tests.
func WithNow(now func() time.Time) LoaderOption {
	return func(loader *Loader) {
		if now != nil {
			loader.now = now
		}
	}
}

// WithGitCacheDir sets the base directory for cached git clones.
func WithGitCacheDir(path string) LoaderOption {
	return func(loader *Loader) {
		if path != "" {
			loader.gitCache = path
		}
	}
}

// LoadFromFilesystem scans a file or directory for workflow YAML files.
func (loader *Loader) LoadFromFilesystem(
	ctx context.Context,
	root string,
) (WorkflowIndex, error) {
	if root == "" {
		return WorkflowIndex{}, fmt.Errorf("filesystem root is required")
	}

	stat, err := os.Stat(root)
	if err != nil {
		return WorkflowIndex{}, fmt.Errorf("stat filesystem root: %w", err)
	}

	root = filepath.Clean(root)
	cacheKey := fmt.Sprintf("filesystem:%s", root)
	if stat.Mode().IsRegular() {
		return loader.loadSingleFile(ctx, cacheKey, root, stat)
	}

	source := SourceMetadata{
		Kind:     SourceFilesystem,
		Location: root,
	}
	return loader.loadDirectory(ctx, cacheKey, root, root, source)
}

// LoadFromURL fetches a workflow from an HTTP endpoint.
func (loader *Loader) LoadFromURL(
	ctx context.Context,
	location string,
) (WorkflowIndex, error) {
	if location == "" {
		return WorkflowIndex{}, fmt.Errorf("url is required")
	}
	if _, err := url.Parse(location); err != nil {
		return WorkflowIndex{}, fmt.Errorf("parse url: %w", err)
	}

	cacheKey := fmt.Sprintf("url:%s", location)
	cached, ok := loader.cache.Get(cacheKey)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return WorkflowIndex{}, fmt.Errorf("build url request: %w", err)
	}
	if ok {
		if cached.Snapshot.ETag != "" {
			request.Header.Set("If-None-Match", cached.Snapshot.ETag)
		}
		if !cached.Snapshot.LastModified.IsZero() {
			request.Header.Set(
				"If-Modified-Since",
				cached.Snapshot.LastModified.UTC().Format(http.TimeFormat),
			)
		}
	}

	response, err := loader.httpClient.Do(request)
	if err != nil {
		return WorkflowIndex{}, fmt.Errorf("fetch url: %w", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			loader.logger.Warn(
				"close workflow url response",
				"error",
				err,
			)
		}
	}()

	if response.StatusCode == http.StatusNotModified && ok {
		return cached, nil
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= 300 {
		return WorkflowIndex{}, fmt.Errorf("url status %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return WorkflowIndex{}, fmt.Errorf("read url body: %w", err)
	}

	workflow := workflowFromContent(SourceMetadata{
		Kind:     SourceURL,
		Location: location,
	}, filepath.Base(request.URL.Path), location, "", body)

	index := WorkflowIndex{
		Source: workflow.Source,
		Snapshot: SourceSnapshot{
			ETag:         response.Header.Get("ETag"),
			LastModified: parseHTTPTime(response.Header.Get("Last-Modified")),
		},
		Workflows: []Workflow{workflow},
		FetchedAt: loader.now().UTC(),
	}
	loader.cache.Set(cacheKey, index)

	return index, nil
}

// LoadFromGit loads workflows from a git repository and optional subdirectory.
func (loader *Loader) LoadFromGit(
	ctx context.Context,
	repoURL string,
	ref string,
	subdir string,
) (WorkflowIndex, error) {
	if repoURL == "" {
		return WorkflowIndex{}, fmt.Errorf("git repo url is required")
	}

	cacheKey := fmt.Sprintf("git:%s:%s:%s", repoURL, ref, subdir)
	repoDir := filepath.Join(loader.gitCache, hashString(cacheKey))
	commit, err := loader.gitClient.Sync(ctx, repoURL, repoDir, ref)
	if err != nil {
		return WorkflowIndex{}, err
	}

	if cached, ok := loader.cache.Get(cacheKey); ok {
		if cached.Snapshot.Commit == commit {
			return cached, nil
		}
	}

	scanRoot := repoDir
	if subdir != "" {
		scanRoot = filepath.Join(repoDir, subdir)
	}
	source := SourceMetadata{
		Kind:     SourceGit,
		Location: repoURL,
		Ref:      ref,
		Subdir:   subdir,
	}
	index, err := loader.loadDirectory(ctx, cacheKey, scanRoot, repoDir, source)
	if err != nil {
		return WorkflowIndex{}, err
	}
	index.Source = source
	index.Snapshot.Commit = commit
	loader.cache.Set(cacheKey, index)

	return index, nil
}

func (loader *Loader) loadSingleFile(
	ctx context.Context,
	cacheKey string,
	path string,
	info os.FileInfo,
) (WorkflowIndex, error) {
	if ctx.Err() != nil {
		return WorkflowIndex{}, ctx.Err()
	}
	if !isWorkflowFile(path) {
		return WorkflowIndex{}, fmt.Errorf("not a workflow file: %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return WorkflowIndex{}, fmt.Errorf("read workflow file: %w", err)
	}

	workflow := workflowFromContent(SourceMetadata{
		Kind:     SourceFilesystem,
		Location: path,
	}, filepath.Base(path), path, path, content)
	workflow.SizeBytes = info.Size()
	workflow.ModifiedAt = info.ModTime().UTC()

	index := WorkflowIndex{
		Source: workflow.Source,
		Snapshot: SourceSnapshot{
			FilesystemFingerprint: fingerprintFile(info, path),
		},
		Workflows: []Workflow{workflow},
		FetchedAt: loader.now().UTC(),
	}

	if cached, ok := loader.cache.Get(cacheKey); ok {
		if cached.Snapshot.FilesystemFingerprint == index.Snapshot.FilesystemFingerprint {
			return cached, nil
		}
	}

	loader.cache.Set(cacheKey, index)
	return index, nil
}

func (loader *Loader) loadDirectory(
	ctx context.Context,
	cacheKey string,
	scanRoot string,
	relativeBase string,
	source SourceMetadata,
) (WorkflowIndex, error) {
	if ctx.Err() != nil {
		return WorkflowIndex{}, ctx.Err()
	}

	fingerprint, workflows, err := scanDirectory(ctx, scanRoot, relativeBase, source)
	if err != nil {
		return WorkflowIndex{}, err
	}

	index := WorkflowIndex{
		Source: source,
		Snapshot: SourceSnapshot{
			FilesystemFingerprint: fingerprint,
		},
		Workflows: workflows,
		FetchedAt: loader.now().UTC(),
	}

	if cached, ok := loader.cache.Get(cacheKey); ok {
		if cached.Snapshot.FilesystemFingerprint == fingerprint {
			return cached, nil
		}
	}

	loader.cache.Set(cacheKey, index)
	return index, nil
}

func scanDirectory(
	ctx context.Context,
	scanRoot string,
	relativeBase string,
	source SourceMetadata,
) (string, []Workflow, error) {
	var workflows []Workflow
	fingerprintHasher := sha256.New()

	err := filepath.WalkDir(scanRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if entry.IsDir() {
			return nil
		}
		if !isWorkflowFile(path) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(relativeBase, path)
		if err != nil {
			return err
		}

		recordFingerprint(fingerprintHasher, relative, info)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read workflow file: %w", err)
		}

		workflow := workflowFromContent(
			source,
			filepath.Base(path),
			relative,
			path,
			content,
		)
		workflow.SizeBytes = info.Size()
		workflow.ModifiedAt = info.ModTime().UTC()
		workflows = append(workflows, workflow)
		return nil
	})

	if err != nil {
		return "", nil, fmt.Errorf("scan workflows: %w", err)
	}

	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].Path < workflows[j].Path
	})

	return hashBytes(fingerprintHasher.Sum(nil)), workflows, nil
}

func isWorkflowFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml" || ext == ".json"
}

func workflowFromContent(
	source SourceMetadata,
	name string,
	path string,
	localPath string,
	content []byte,
) Workflow {
	contentHash := sha256.Sum256(content)
	workflow := Workflow{
		Name:          strings.TrimSuffix(name, filepath.Ext(name)),
		Source:        source,
		Path:          path,
		LocalPath:     localPath,
		ContentSHA256: hex.EncodeToString(contentHash[:]),
		Content:       append([]byte(nil), content...),
	}
	workflow.ID = workflowID(workflow)
	return workflow
}

func workflowID(workflow Workflow) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(string(workflow.Source.Kind)))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(workflow.Source.Location))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(workflow.Path))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(workflow.ContentSHA256))
	return hashBytes(hasher.Sum(nil))
}

func fingerprintFile(info os.FileInfo, path string) string {
	hasher := sha256.New()
	recordFingerprint(hasher, filepath.Base(path), info)
	return hashBytes(hasher.Sum(nil))
}

func recordFingerprint(hasher hashWriter, path string, info os.FileInfo) {
	_, _ = hasher.Write([]byte(path))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(fmt.Sprintf("%d", info.Size())))
	_, _ = hasher.Write([]byte(":"))
	_, _ = hasher.Write([]byte(fmt.Sprintf("%d", info.ModTime().UTC().UnixNano())))
}

type hashWriter interface {
	Write(p []byte) (n int, err error)
}

func hashBytes(data []byte) string {
	return hex.EncodeToString(data)
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func parseHTTPTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(http.TimeFormat, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

// Package workflow provides workflow discovery and loading helpers.
package workflow

import "time"

// SourceKind identifies where a workflow was loaded from.
type SourceKind string

const (
	// SourceFilesystem indicates workflows loaded from local files.
	SourceFilesystem SourceKind = "filesystem"
	// SourceURL indicates a workflow loaded from an HTTP endpoint.
	SourceURL SourceKind = "url"
	// SourceGit indicates workflows loaded from a git repository.
	SourceGit SourceKind = "git"
)

// SourceMetadata describes the workflow source location.
type SourceMetadata struct {
	Kind     SourceKind
	Location string
	Ref      string
	Subdir   string
}

// SourceSnapshot records source-specific cache metadata.
type SourceSnapshot struct {
	FilesystemFingerprint string
	ETag                  string
	LastModified          time.Time
	Commit                string
}

// Workflow captures a discovered workflow and its metadata.
type Workflow struct {
	ID            string
	Name          string
	Source        SourceMetadata
	Path          string
	LocalPath     string
	SizeBytes     int64
	ModifiedAt    time.Time
	ContentSHA256 string
	Content       []byte
}

// WorkflowIndex contains workflows discovered from a single source.
type WorkflowIndex struct {
	Source    SourceMetadata
	Snapshot  SourceSnapshot
	Workflows []Workflow
	FetchedAt time.Time
}

package workflow

import "time"

// CacheStatus describes whether a discovery request used cached data.
type CacheStatus string

const (
	// CacheStatusHit indicates cached data was reused.
	CacheStatusHit CacheStatus = "hit"
	// CacheStatusMiss indicates cached data was not reused.
	CacheStatusMiss CacheStatus = "miss"
)

// ProgressStage identifies a loader milestone for progress reporting.
type ProgressStage string

const (
	// ProgressStageFetch indicates remote fetch activity.
	ProgressStageFetch ProgressStage = "fetch"
	// ProgressStageCheckout indicates a git checkout step.
	ProgressStageCheckout ProgressStage = "checkout"
	// ProgressStageScan indicates filesystem scan activity.
	ProgressStageScan ProgressStage = "scan"
)

// ProgressEvent captures a loader milestone with timestamps and outcome.
type ProgressEvent struct {
	Stage       ProgressStage
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	Error       string
}

// LoadTiming captures elapsed durations for discovery milestones.
type LoadTiming struct {
	Total    time.Duration
	Fetch    time.Duration
	Checkout time.Duration
	Scan     time.Duration
}

// LoadDetails records discovery output metadata for a workflow source.
type LoadDetails struct {
	Index       WorkflowIndex
	CacheStatus CacheStatus
	Timing      LoadTiming
	Progress    []ProgressEvent
}

// ProgressReporter receives progress milestones as they complete.
type ProgressReporter interface {
	Start(stage ProgressStage)
	Finish(stage ProgressStage, err error)
}

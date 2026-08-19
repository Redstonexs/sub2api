package usagestats

import "time"

// AccountWindowStatsRequest describes a single requested usage window for a
// specific account. Each request carries its own start time so one batch can
// span heterogeneous windows (e.g. an Anthropic 5h session window, an Anthropic
// rolling 7d window, and an OpenAI codex 5h/7d window) in a single query.
type AccountWindowStatsRequest struct {
	AccountID int64
	// WindowKey is a logical window identifier (e.g. "5h" / "7d") used to
	// correlate the result back to the caller's window.
	WindowKey string
	// StartTime is the individual window start used for the aggregation.
	StartTime time.Time
}

// AccountWindowStatsKey identifies a single window result within a batch.
type AccountWindowStatsKey struct {
	AccountID int64
	WindowKey string
}

package domain

// Group QoS metric values decide which cost figure accumulates toward a tier
// threshold. "list" counts undiscounted cost so thresholds stay comparable
// across groups with different rate multipliers; "charged" counts what the user
// actually paid.
const (
	GroupQoSMetricList    = "list"
	GroupQoSMetricCharged = "charged"
)

// Group QoS windows reuse the rolling windows that already drive the group
// daily/weekly/monthly USD limits.
const (
	GroupQoSWindowDaily   = "daily"
	GroupQoSWindowWeekly  = "weekly"
	GroupQoSWindowMonthly = "monthly"
)

// GroupQoSModelMapping reroutes one requested model to a cheaper one once a QoS
// tier is in effect. From supports a single trailing "*" wildcard.
type GroupQoSModelMapping struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// GroupQoSTier is one rung of a group's degradation ladder. It applies once the
// requester's usage in Window reaches ThresholdUSD. Every action is optional; a
// tier may apply any subset of them.
type GroupQoSTier struct {
	Window       string  `json:"window"`
	ThresholdUSD float64 `json:"threshold_usd"`

	ModelMappings      []GroupQoSModelMapping `json:"model_mappings,omitempty"`
	MaxReasoningEffort string                 `json:"max_reasoning_effort,omitempty"`
	// RPMLimit caps requests per minute while the tier is active. nil leaves the
	// existing Override → Group → User cascade untouched; 0 means unlimited.
	RPMLimit *int `json:"rpm_limit,omitempty"`
	// Block rejects the request outright — the final rung of the ladder.
	Block bool `json:"block,omitempty"`
}

package model

type RunRecoveryDetail struct {
	RunID                  string `json:"runId"`
	ExperimentID           string `json:"experimentId"`
	RunStatus              string `json:"runStatus"`
	FailureReason          string `json:"failureReason"`
	FailureStage           string `json:"failureStage"`
	LastLogSummary         string `json:"lastLogSummary"`
	SuggestRetry           bool   `json:"suggestRetry"`
	RetryCount             int    `json:"retryCount"`
	LatestAssignedServerID string `json:"latestAssignedServerId,omitempty"`
}

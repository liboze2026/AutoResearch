package model

import "time"

type ExperimentQueueResult struct {
	ExperimentID string        `json:"experimentId"`
	Run          ExperimentRun `json:"run"`
}

type SchedulerCandidate struct {
	ServerID         string     `json:"serverId"`
	ServerName       string     `json:"serverName"`
	HeartbeatAt      time.Time  `json:"heartbeatAt"`
	Status           string     `json:"status"`
	BestGPUIndex     int        `json:"bestGpuIndex"`
	BestGPUName      string     `json:"bestGpuName"`
	BestFreeMemMB    int        `json:"bestFreeMemMb"`
	BestUtilization  int        `json:"bestUtilization"`
	QueueLength      int        `json:"queueLength"`
	SnapshotCaptured *time.Time `json:"snapshotCaptured,omitempty"`
	Eligible         bool       `json:"eligible"`
	IneligibleReason string     `json:"ineligibleReason,omitempty"`
}

type ScheduleResult struct {
	Run      ExperimentRun      `json:"run"`
	Decision SchedulerDecision  `json:"decision"`
	Chosen   SchedulerCandidate `json:"chosen"`
}

package config

import (
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	Env                           string
	Port                          string
	PostgresDSN                   string
	PythonServiceURL              string
	PythonExec                    string
	PythonAgentsDir               string
	PythonTemplatesDir            string
	PythonRunnersDir              string
	WorkspaceRoot                 string
	SSHBinary                     string
	SSHClientMode                 string
	SSHDialTimeoutSec             int
	SSHCommandTimeoutSec          int
	RemoteExecutionMode           string
	RemoteWorkRoot                string
	Phase4RemoteWorkRoot          string
	RemoteRunnerEntrypoint        string
	RemoteDatasetRunnerEntrypoint string
	DatasetScanMode               string
	DatasetIndexMode              string
	DatasetPreviewLimit           int
	OverviewStatsMode             string
	OverviewTrendDays             int
	ServerHeartbeatIntervalSec    int
	GPUSnapshotIntervalSec        int
}

func Load() AppConfig {
	dialTimeout, _ := strconv.Atoi(getEnv("SSH_DIAL_TIMEOUT_SEC", "4"))
	if dialTimeout <= 0 {
		dialTimeout = 4
	}

	commandTimeout, _ := strconv.Atoi(getEnv("SSH_COMMAND_TIMEOUT_SEC", "20"))
	if commandTimeout <= 0 {
		commandTimeout = 20
	}

	previewLimit, _ := strconv.Atoi(getEnv("DATASET_PREVIEW_LIMIT", "12"))
	if previewLimit <= 0 {
		previewLimit = 12
	}

	trendDays, _ := strconv.Atoi(getEnv("OVERVIEW_TREND_DAYS", "7"))
	if trendDays <= 0 {
		trendDays = 7
	}

	heartbeatInterval, _ := strconv.Atoi(getEnv("SERVER_HEARTBEAT_INTERVAL_SEC", "0"))
	if heartbeatInterval < 0 {
		heartbeatInterval = 0
	}

	gpuSnapshotInterval, _ := strconv.Atoi(getEnv("GPU_SNAPSHOT_INTERVAL_SEC", "0"))
	if gpuSnapshotInterval < 0 {
		gpuSnapshotInterval = 0
	}

	return AppConfig{
		Env:                           getEnv("APP_ENV", "dev"),
		Port:                          getEnv("APP_PORT", "8080"),
		PostgresDSN:                   getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/mrag_platform?sslmode=disable"),
		PythonServiceURL:              getEnv("PYTHON_SERVICE_URL", "http://localhost:8000"),
		PythonExec:                    getEnv("PYTHON_EXEC", "python"),
		PythonAgentsDir:               getEnv("PYTHON_AGENTS_DIR", "../python_agents"),
		PythonTemplatesDir:            getEnv("PYTHON_TEMPLATES_DIR", "../python_templates"),
		PythonRunnersDir:              getEnv("PYTHON_RUNNERS_DIR", "../python_runners"),
		WorkspaceRoot:                 getEnv("WORKSPACE_ROOT", "workspace"),
		SSHBinary:                     getEnv("SSH_BINARY", "ssh"),
		SSHClientMode:                 normalizeModeValue(getEnv("SSH_CLIENT_MODE", "real")),
		SSHDialTimeoutSec:             dialTimeout,
		SSHCommandTimeoutSec:          commandTimeout,
		RemoteExecutionMode:           normalizeModeValue(getEnv("REMOTE_EXECUTION_MODE", "real")),
		RemoteWorkRoot:                getEnv("REMOTE_WORK_ROOT", "/home/bzli/lbz"),
		Phase4RemoteWorkRoot:          getEnv("PHASE4_REMOTE_WORK_ROOT", "/home/bzli/mrag"),
		RemoteRunnerEntrypoint:        getEnv("REMOTE_RUNNER_ENTRYPOINT", "./bin/mrag-remote-runner"),
		RemoteDatasetRunnerEntrypoint: getEnv("REMOTE_DATASET_RUNNER_ENTRYPOINT", "./bin/mrag-dataset-runner"),
		DatasetScanMode:               normalizeModeValue(getEnvAny([]string{"DATASET_SCAN_MODE", "DATASET_REMOTE_MODE", "DATASET_LOCAL_MODE"}, "real")),
		DatasetIndexMode:              normalizeModeValue(getEnvAny([]string{"DATASET_INDEX_MODE", "DATASET_REMOTE_MODE", "DATASET_LOCAL_MODE"}, "real")),
		DatasetPreviewLimit:           previewLimit,
		OverviewStatsMode:             normalizeModeValue(getEnv("OVERVIEW_STATS_MODE", "real")),
		OverviewTrendDays:             trendDays,
		ServerHeartbeatIntervalSec:    heartbeatInterval,
		GPUSnapshotIntervalSec:        gpuSnapshotInterval,
	}
}

func getEnv(k string, dv string) string {
	v := os.Getenv(k)
	if v == "" {
		return dv
	}
	return v
}

func getEnvAny(keys []string, dv string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return dv
}

func normalizeModeValue(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "mock") {
		return "mock"
	}
	return "real"
}

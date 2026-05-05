package model

import "time"

type Dataset struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	Tags             []string         `json:"tags"`
	SourceType       string           `json:"sourceType"`
	Modality         string           `json:"modality"`
	Version          string           `json:"version"`
	Size             string           `json:"size"`
	Samples          int64            `json:"samples"`
	Description      string           `json:"description"`
	Path             string           `json:"path"`
	ServerID         string           `json:"serverId,omitempty"`
	ServerName       string           `json:"serverName,omitempty"`
	IndexStatus      string           `json:"indexStatus"`
	FileCount        int64            `json:"fileCount"`
	DirectoryCount   int64            `json:"directoryCount"`
	TotalSizeBytes   int64            `json:"totalSizeBytes"`
	FileTypes        map[string]int64 `json:"fileTypes,omitempty"`
	DetectedModality string           `json:"detectedModality,omitempty"`
	LastScanStatus   string           `json:"lastScanStatus"`
	LastScanAt       *time.Time       `json:"lastScanAt,omitempty"`
	LastModifiedAt   *time.Time       `json:"lastModifiedAt,omitempty"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type DatasetImportRequest struct {
	Name        string   `json:"name" binding:"required"`
	SourceType  string   `json:"sourceType" binding:"required"`
	Path        string   `json:"path" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Tags        []string `json:"tags"`
	Modality    string   `json:"modality"`
	Version     string   `json:"version"`
	ServerID    string   `json:"serverId"`
}

type DatasetUpdateRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description" binding:"required"`
	Tags        []string `json:"tags"`
	Modality    string   `json:"modality"`
	Version     string   `json:"version"`
}

type DatasetPathValidationRequest struct {
	SourceType string `json:"sourceType" binding:"required"`
	Path       string `json:"path" binding:"required"`
	ServerID   string `json:"serverId"`
}

type DatasetPathValidationResult struct {
	SourceType  string    `json:"sourceType"`
	Path        string    `json:"path"`
	ServerID    string    `json:"serverId,omitempty"`
	ServerName  string    `json:"serverName,omitempty"`
	Mode        string    `json:"mode"`
	Valid       bool      `json:"valid"`
	Exists      bool      `json:"exists"`
	IsDirectory bool      `json:"isDirectory"`
	ErrorType   string    `json:"errorType,omitempty"`
	Message     string    `json:"message"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type ServerDatasetScanRequest struct {
	RootPath string `json:"rootPath"`
	MaxDepth int    `json:"maxDepth"`
}

type ServerDatasetCandidate struct {
	Name           string     `json:"name"`
	Path           string     `json:"path"`
	Size           string     `json:"size"`
	TotalSizeBytes int64      `json:"totalSizeBytes"`
	FileCount      int64      `json:"fileCount"`
	DirectoryCount int64      `json:"directoryCount"`
	LastModifiedAt *time.Time `json:"lastModifiedAt,omitempty"`
	Modality       string     `json:"modality,omitempty"`
	Status         string     `json:"status,omitempty"`
	Description    string     `json:"description,omitempty"`
}

type ServerDatasetScanResult struct {
	ServerID   string                   `json:"serverId"`
	ServerName string                   `json:"serverName,omitempty"`
	Mode       string                   `json:"mode"`
	RootPath   string                   `json:"rootPath"`
	ScannedAt  time.Time                `json:"scannedAt"`
	Candidates []ServerDatasetCandidate `json:"candidates"`
}

type DatasetHierarchySummaryItem struct {
	Level     int    `json:"level"`
	Path      string `json:"path"`
	ItemCount int64  `json:"itemCount"`
}

type DatasetPreviewItem struct {
	ID           int64  `json:"id"`
	ScanRecordID string `json:"scanRecordId,omitempty"`
	Name         string `json:"name"`
	ItemType     string `json:"itemType"`
	Category     string `json:"category"`
	RelativePath string `json:"relativePath"`
	SizeBytes    int64  `json:"sizeBytes"`
	Depth        int    `json:"depth"`
}

type DatasetScanRecord struct {
	ID               string                        `json:"id"`
	DatasetID        string                        `json:"datasetId"`
	ServerID         string                        `json:"serverId,omitempty"`
	RuntimeMode      string                        `json:"runtimeMode"`
	ScanStatus       string                        `json:"scanStatus"`
	ValidationStatus string                        `json:"validationStatus"`
	RootPath         string                        `json:"rootPath"`
	FileCount        int64                         `json:"fileCount"`
	DirectoryCount   int64                         `json:"directoryCount"`
	TotalSizeBytes   int64                         `json:"totalSizeBytes"`
	FileTypes        map[string]int64              `json:"fileTypes"`
	HierarchySummary []DatasetHierarchySummaryItem `json:"hierarchySummary"`
	InferredModality string                        `json:"inferredModality"`
	RecentModifiedAt *time.Time                    `json:"recentModifiedAt,omitempty"`
	ScannedAt        time.Time                     `json:"scannedAt"`
	ErrorMessage     string                        `json:"errorMessage,omitempty"`
}

type DatasetIndexTask struct {
	ID              string                 `json:"id"`
	DatasetID       string                 `json:"datasetId"`
	ServerID        string                 `json:"serverId,omitempty"`
	SourceType      string                 `json:"sourceType"`
	ExecutorMode    string                 `json:"executorMode"`
	Status          string                 `json:"status"`
	RemoteTaskID    string                 `json:"remoteTaskId,omitempty"`
	LogPath         string                 `json:"logPath,omitempty"`
	StatusPath      string                 `json:"statusPath,omitempty"`
	ResultPath      string                 `json:"resultPath,omitempty"`
	ErrorMessage    string                 `json:"errorMessage,omitempty"`
	RequestPayload  map[string]interface{} `json:"requestPayload,omitempty"`
	ResponsePayload map[string]interface{} `json:"responsePayload,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	FinishedAt      *time.Time             `json:"finishedAt,omitempty"`
	Logs            []DatasetIndexTaskLog  `json:"logs,omitempty"`
}

type DatasetIndexTaskLog struct {
	ID        int64     `json:"id"`
	TaskID    string    `json:"taskId"`
	Level     string    `json:"level"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type DatasetDetail struct {
	Dataset         Dataset              `json:"dataset"`
	LatestScan      *DatasetScanRecord   `json:"latestScan,omitempty"`
	PreviewItems    []DatasetPreviewItem `json:"previewItems"`
	LatestIndexTask *DatasetIndexTask    `json:"latestIndexTask,omitempty"`
	IndexTasks      []DatasetIndexTask   `json:"indexTasks"`
}

type Server struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Host           string                 `json:"host"`
	SSHPort        int                    `json:"sshPort"`
	Username       string                 `json:"username"`
	AuthType       string                 `json:"authType"`
	HasPassword    bool                   `json:"hasPassword,omitempty"`
	Password       string                 `json:"password,omitempty"`
	PrivateKey     string                 `json:"privateKey,omitempty"`
	Status         string                 `json:"status"`
	StatusMessage  string                 `json:"statusMessage,omitempty"`
	GPUInfo        string                 `json:"gpuInfo"`
	RemoteRoot     string                 `json:"remoteRoot"`
	TaskWorkdir    string                 `json:"taskWorkdir"`
	Config         map[string]interface{} `json:"config,omitempty"`
	AvailableGPUs  int                    `json:"availableGpus,omitempty"`
	TotalGPUs      int                    `json:"totalGpus,omitempty"`
	LastHeartbeat  *time.Time             `json:"lastHeartbeat"`
	LastGPUCheckAt *time.Time             `json:"lastGpuCheckAt,omitempty"`
}

type ServerConnectionTestResult struct {
	ServerID   string    `json:"serverId"`
	ServerName string    `json:"serverName"`
	Mode       string    `json:"mode"`
	Target     string    `json:"target"`
	Result     string    `json:"result"`
	Reachable  bool      `json:"reachable"`
	Message    string    `json:"message"`
	RemoteHost string    `json:"remoteHost,omitempty"`
	RemoteUser string    `json:"remoteUser,omitempty"`
	Stdout     string    `json:"stdout,omitempty"`
	Stderr     string    `json:"stderr,omitempty"`
	ExitCode   int       `json:"exitCode"`
	LatencyMs  int64     `json:"latencyMs"`
	CheckedAt  time.Time `json:"checkedAt"`
}

type GPUDeviceStatus struct {
	Index         int    `json:"index"`
	Name          string `json:"name"`
	MemoryUsedMB  int64  `json:"memoryUsedMb,omitempty"`
	MemoryTotalMB int64  `json:"memoryTotalMb,omitempty"`
	Utilization   int64  `json:"utilization,omitempty"`
	Processes     int64  `json:"processes,omitempty"`
	Available     bool   `json:"available"`
}

type GPUProbeResult struct {
	ServerID          string            `json:"serverId"`
	ServerName        string            `json:"serverName"`
	Mode              string            `json:"mode"`
	Summary           string            `json:"summary"`
	AvailableGPUCount int               `json:"availableGpuCount"`
	TotalGPUCount     int               `json:"totalGpuCount"`
	CheckedAt         time.Time         `json:"checkedAt"`
	Devices           []GPUDeviceStatus `json:"devices"`
}

type ServerStatusSnapshot struct {
	ServerID  string    `json:"serverId"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checkedAt"`
}

type OverviewTrendPoint struct {
	Date          string `json:"date"`
	Datasets      int64  `json:"datasets"`
	Scanned       int64  `json:"scanned"`
	OnlineServers int64  `json:"onlineServers"`
}

type RuntimeModeItem struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Mode         string `json:"mode"`
	Summary      string `json:"summary"`
	RealBehavior string `json:"realBehavior"`
	MockBehavior string `json:"mockBehavior"`
}

type RuntimeProfile struct {
	Preset               string            `json:"preset"`
	GeneratedAt          time.Time         `json:"generatedAt"`
	RemoteExecutionMode  string            `json:"remoteExecutionMode"`
	DatasetScanMode      string            `json:"datasetScanMode"`
	DatasetIndexMode     string            `json:"datasetIndexMode"`
	OverviewStatsMode    string            `json:"overviewStatsMode"`
	ServerConnectionMode string            `json:"serverConnectionMode"`
	Modes                []RuntimeModeItem `json:"modes"`
	Notes                []string          `json:"notes"`
}

type OverviewStats struct {
	PlatformIntro    string               `json:"platformIntro"`
	StatsMode        string               `json:"statsMode"`
	StatsGeneratedAt time.Time            `json:"statsGeneratedAt"`
	DatasetCount     int64                `json:"datasetCount"`
	ScannedDatasets  int64                `json:"scannedDatasets"`
	PendingIndexes   int64                `json:"pendingIndexes"`
	ServerOnline     int64                `json:"serverOnline"`
	ServerTotal      int64                `json:"serverTotal"`
	Trend            []OverviewTrendPoint `json:"trend"`
	Notes            []string             `json:"notes"`
}

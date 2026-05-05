export type DataSourceType = "local" | "remote";
export type DatasetModality = "text" | "image" | "audio" | "video" | "multimodal";
export type IndexStatus = "none" | "ready" | "building" | "failed";

export interface Dataset {
  id: string;
  name: string;
  tags: string[];
  sourceType: DataSourceType;
  modality: DatasetModality;
  version: string;
  size: string;
  samples: number;
  description: string;
  path: string;
  serverId?: string;
  serverName?: string;
  indexStatus: IndexStatus;
  fileCount: number;
  directoryCount: number;
  totalSizeBytes: number;
  fileTypes: Record<string, number>;
  detectedModality?: string;
  lastScanStatus: string;
  lastScanAt?: string;
  lastModifiedAt?: string;
  updatedAt: string;
}

export interface DatasetImportRequest {
  name: string;
  sourceType: DataSourceType;
  path: string;
  description: string;
  tags: string[];
  modality?: DatasetModality;
  version?: string;
  serverId?: string;
}

export interface DatasetUpdateRequest {
  name: string;
  description: string;
  tags: string[];
  modality?: DatasetModality;
  version?: string;
}

export interface DatasetPathValidationRequest {
  sourceType: DataSourceType;
  path: string;
  serverId?: string;
}

export interface DatasetPathValidationResult {
  sourceType: DataSourceType;
  path: string;
  serverId?: string;
  serverName?: string;
  mode: "mock" | "real" | string;
  valid: boolean;
  exists: boolean;
  isDirectory: boolean;
  errorType?: "not_found" | "permission_denied" | "not_directory" | string;
  message: string;
  checkedAt: string;
}

export interface ServerDatasetCandidate {
  name: string;
  path: string;
  size: string;
  totalSizeBytes: number;
  fileCount: number;
  directoryCount: number;
  lastModifiedAt?: string;
  modality?: string;
  status?: "new" | "registered" | "invalid" | string;
  description?: string;
}

export interface ServerDatasetScanRequest {
  serverId: string;
  rootPath?: string;
  maxDepth?: number;
}

export interface ServerDatasetScanResult {
  serverId: string;
  serverName?: string;
  mode: "mock" | "real" | string;
  rootPath: string;
  scannedAt: string;
  candidates: ServerDatasetCandidate[];
}

export interface DatasetHierarchySummaryItem {
  level: number;
  path: string;
  itemCount: number;
}

export interface DatasetPreviewItem {
  id: number;
  scanRecordId?: string;
  name: string;
  itemType: "file" | "directory" | string;
  category: string;
  relativePath: string;
  sizeBytes: number;
  depth: number;
}

export interface DatasetScanRecord {
  id: string;
  datasetId: string;
  serverId?: string;
  runtimeMode: "mock" | "real" | string;
  scanStatus: string;
  validationStatus: string;
  rootPath: string;
  fileCount: number;
  directoryCount: number;
  totalSizeBytes: number;
  fileTypes: Record<string, number>;
  hierarchySummary: DatasetHierarchySummaryItem[];
  inferredModality: string;
  recentModifiedAt?: string;
  scannedAt: string;
  errorMessage?: string;
}

export interface DatasetIndexTaskLog {
  id: number;
  taskId: string;
  level: string;
  content: string;
  createdAt: string;
}

export interface DatasetIndexTask {
  id: string;
  datasetId: string;
  serverId?: string;
  sourceType: DataSourceType;
  executorMode: "mock" | "real" | string;
  status: string;
  remoteTaskId?: string;
  logPath?: string;
  statusPath?: string;
  resultPath?: string;
  errorMessage?: string;
  requestPayload?: Record<string, unknown>;
  responsePayload?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  finishedAt?: string;
  logs: DatasetIndexTaskLog[];
}

export interface DatasetDetail {
  dataset: Dataset;
  latestScan?: DatasetScanRecord;
  previewItems: DatasetPreviewItem[];
  latestIndexTask?: DatasetIndexTask;
  indexTasks: DatasetIndexTask[];
}

export interface ServerNode {
  id: string;
  name: string;
  host: string;
  sshPort: number;
  username: string;
  authType: string;
  hasPassword?: boolean;
  status: "online" | "offline" | "connecting";
  gpuInfo: string;
  remoteRoot: string;
  taskWorkdir: string;
  lastHeartbeat: string | null;
  config?: Record<string, unknown>;
  statusMessage?: string;
  availableGpus?: number;
  totalGpus?: number;
  lastGpuCheckAt?: string | null;
}

export interface ServerNodePayload {
  name: string;
  host: string;
  sshPort: number;
  username: string;
  authType: string;
  password?: string;
  remoteRoot: string;
  taskWorkdir: string;
  config: Record<string, unknown>;
}

export interface SSHConnectionTestResult {
  serverId: string;
  serverName: string;
  mode: string;
  target: string;
  result: "login_success" | "host_unreachable" | "handshake_failed" | "auth_failed" | string;
  reachable: boolean;
  message: string;
  remoteHost?: string;
  remoteUser?: string;
  stdout?: string;
  stderr?: string;
  exitCode: number;
  latencyMs: number;
  checkedAt: string;
}

export interface GPUDeviceStatus {
  index: number;
  name: string;
  memoryUsedMb?: number;
  memoryTotalMb?: number;
  utilization?: number;
  processes?: number;
  available: boolean;
}

export interface GPUProbeResult {
  serverId: string;
  serverName: string;
  mode: "mock" | "real" | string;
  summary: string;
  availableGpuCount: number;
  totalGpuCount: number;
  checkedAt: string;
  devices: GPUDeviceStatus[];
}

export interface ServerStatusSnapshot {
  serverId: string;
  status: ServerNode["status"];
  message: string;
  checkedAt: string;
}

export interface RuntimeModeItem {
  key: string;
  label: string;
  mode: "mock" | "real" | string;
  summary: string;
  realBehavior: string;
  mockBehavior: string;
}

export interface RuntimeProfile {
  preset: "all-real" | "all-mock" | "mixed" | string;
  generatedAt: string;
  remoteExecutionMode: "mock" | "real" | string;
  datasetScanMode: "mock" | "real" | string;
  datasetIndexMode: "mock" | "real" | string;
  overviewStatsMode: "mock" | "real" | string;
  serverConnectionMode: "mock" | "real" | string;
  modes: RuntimeModeItem[];
  notes: string[];
}

export interface OverviewTrendPoint {
  date: string;
  datasets?: number;
  scanned?: number;
  onlineServers?: number;
}

export interface OverviewStats {
  platformIntro: string;
  statsMode: string;
  statsGeneratedAt: string;
  datasetCount: number;
  scannedDatasets?: number;
  pendingIndexes?: number;
  serverOnline: number;
  serverTotal: number;
  trend: OverviewTrendPoint[];
  notes: string[];
}

export type PaperStatus = "imported" | "parsed" | "insight_extracted" | string;
export type IdeaStatus = "draft" | "shortlisted" | "archived" | string;
export type IdeaSourceType = "auto" | "human" | "mixed" | string;
export type DatasetAssetStatus = "draft" | "active" | "archived" | string;
export type BaselineSourceType = "manual" | "result_archive" | "mixed" | string;
export type ResultArchiveStatus = "draft" | "archived" | "reviewed" | string;

export interface Paper {
  id: string;
  title: string;
  abstract: string;
  authors: string;
  venue: string;
  year: number;
  status: PaperStatus;
  sourceType: string;
  parseMode?: string;
  parseError?: string;
  parserNote?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PaperFile {
  id: string;
  paperId: string;
  filePath: string;
  fileType: string;
  checksum: string;
  createdAt: string;
  updatedAt: string;
}

export interface PaperInsight {
  id: string;
  paperId: string;
  summaryMd: string;
  contributionsJson?: unknown;
  methodsJson?: unknown;
  limitationsJson?: unknown;
  extractStatus: string;
  extractError?: string;
  createdAt: string;
  updatedAt: string;
}

export interface PaperImportResult {
  paper: Paper;
  files: PaperFile[];
  parseMode: string;
  mockParsed: boolean;
  parserNote: string;
}

export interface PaperDetail {
  paper: Paper;
  files: PaperFile[];
  insightList?: PaperInsight[];
}

export interface PaperParseResult {
  paper: Paper;
  parseMode: string;
  mockParsed: boolean;
  parserNote: string;
}

export interface PaperInsightExtractionResult {
  paperId: string;
  extractMode: string;
  mockExtracted: boolean;
  summaryPath: string;
  insight: PaperInsight;
}

export interface Idea {
  id: string;
  title: string;
  descriptionMd: string;
  status: IdeaStatus;
  weight: number;
  sourceType: IdeaSourceType;
  priority: number;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface IdeaSource {
  id: number;
  ideaId: string;
  paperId?: string;
  paperInsightId?: string;
  sourceNote: string;
  paperTitle?: string;
  createdAt: string;
  updatedAt: string;
}

export interface IdeaCreateRequest {
  title: string;
  descriptionMd: string;
  status?: IdeaStatus;
  weight?: number;
  priority?: number;
  confidence?: number;
  sourceType?: IdeaSourceType;
  sourceNote?: string;
}

export interface IdeaUpdateRequest {
  title?: string;
  descriptionMd?: string;
  status?: IdeaStatus;
  weight?: number;
  priority?: number;
  confidence?: number;
  sourceType?: IdeaSourceType;
}

export interface IdeaDetail {
  idea: Idea;
  sources: IdeaSource[];
}

export interface IdeaGenerationResult {
  paperId: string;
  ideas: Idea[];
}

export interface DatasetAsset {
  id: string;
  name: string;
  descriptionMd: string;
  taskType: string;
  status: DatasetAssetStatus;
  sourceType: string;
  localOrRemotePath: string;
  readmeMd?: string;
  loaderNoteMd?: string;
  schemaNoteMd?: string;
  existingDatasetRef?: string;
  existingDatasetName?: string;
  createdAt: string;
  updatedAt: string;
}

export interface DatasetAssetSource {
  id: number;
  datasetAssetId: string;
  existingDatasetRef: string;
  sourceKind: string;
  existingDatasetName?: string;
  createdAt: string;
  updatedAt: string;
}

export interface DatasetAssetCreateRequest {
  name: string;
  descriptionMd: string;
  taskType: string;
  status?: DatasetAssetStatus;
  sourceType?: string;
  localOrRemotePath: string;
  readmeMd?: string;
  loaderNoteMd?: string;
  schemaNoteMd?: string;
}

export interface DatasetAssetRegisterFromScanRequest {
  existingDatasetRef?: string;
  scanRecordId?: string;
  name?: string;
  descriptionMd?: string;
  taskType?: string;
  status?: DatasetAssetStatus;
  sourceType?: string;
  readmeMd?: string;
  loaderNoteMd?: string;
  schemaNoteMd?: string;
}

export interface DatasetAssetDetail {
  asset: DatasetAsset;
  sources: DatasetAssetSource[];
}

export interface Baseline {
  id: string;
  datasetAssetId: string;
  name: string;
  metricSchemaJson?: Record<string, unknown>;
  resultJson?: Record<string, unknown>;
  noteMd: string;
  sourceType: BaselineSourceType;
  createdAt: string;
  updatedAt: string;
}

export interface BaselineCreateRequest {
  datasetAssetId: string;
  name: string;
  metricSchemaJson?: Record<string, unknown>;
  resultJson?: Record<string, unknown>;
  noteMd?: string;
  sourceType?: BaselineSourceType;
}

export interface BaselineUpdateRequest {
  name?: string;
  metricSchemaJson?: Record<string, unknown>;
  resultJson?: Record<string, unknown>;
  noteMd?: string;
  sourceType?: BaselineSourceType;
}

export interface BaselineDetail {
  baseline: Baseline;
  datasetAsset: DatasetAsset;
}

export interface ArchiveFile {
  id: number;
  archiveId: string;
  filePath: string;
  fileKind: string;
  checksum: string;
  createdAt: string;
  updatedAt: string;
}

export interface ArchiveFileInput {
  fileName: string;
  fileKind: string;
  content: string;
}

export interface ResultArchive {
  id: string;
  title: string;
  datasetAssetId: string;
  baselineId?: string;
  ideaId?: string;
  serverId?: string;
  summaryMd: string;
  metricJson?: Record<string, unknown>;
  status: ResultArchiveStatus;
  noteMd: string;
  createdAt: string;
  updatedAt: string;
}

export interface ResultArchiveCreateRequest {
  title: string;
  datasetAssetId: string;
  baselineId?: string;
  ideaId?: string;
  serverId?: string;
  summaryMd: string;
  metricJson?: Record<string, unknown>;
  status?: ResultArchiveStatus;
  noteMd?: string;
  files?: ArchiveFileInput[];
}

export interface ResultArchiveUpdateRequest {
  title?: string;
  baselineId?: string;
  ideaId?: string;
  serverId?: string;
  summaryMd?: string;
  metricJson?: Record<string, unknown>;
  status?: ResultArchiveStatus;
  noteMd?: string;
  files?: ArchiveFileInput[];
}

export interface ResultArchiveDetail {
  archive: ResultArchive;
  files: ArchiveFile[];
}

export type ExperimentStatus =
  | "draft"
  | "spec_ready"
  | "queued"
  | "scheduled"
  | "running"
  | "succeeded"
  | "failed"
  | "cancelled"
  | "archived"
  | string;

export type ExperimentRunStatus =
  | "queued"
  | "scheduled"
  | "preparing"
  | "running"
  | "succeeded"
  | "failed"
  | string;

export interface Experiment {
  id: string;
  ideaId?: string;
  datasetAssetId: string;
  baselineId?: string;
  title: string;
  status: ExperimentStatus;
  priority: number;
  currentRunId?: string;
  summaryMd: string;
  ownerNoteMd: string;
  createdAt: string;
  updatedAt: string;
}

export interface ExperimentCreateRequest {
  datasetAssetId: string;
  ideaId?: string;
  baselineId?: string;
  title?: string;
  priority?: number;
  summaryMd?: string;
  ownerNoteMd?: string;
}

export interface ExperimentSpec {
  id: string;
  experimentId: string;
  specJson: Record<string, unknown>;
  templateType: string;
  generatedFrom: Record<string, unknown>;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface ExperimentSpecDetail {
  spec: ExperimentSpec;
  workspacePath: string;
  generatorSource: string;
}

export interface ExperimentDetail {
  experiment: Experiment;
  datasetAsset: DatasetAsset;
  idea?: Idea;
  baseline?: Baseline;
  latestSpec?: ExperimentSpec;
}

export interface ExperimentRun {
  id: string;
  experimentId: string;
  specId?: string;
  assignedServerId?: string;
  runStatus: ExperimentRunStatus;
  remoteWorkdir: string;
  remoteJobId?: string;
  startedAt?: string;
  endedAt?: string;
  retryCount: number;
  exitCode?: number;
  resultJson?: Record<string, unknown>;
  errorMessage: string;
  createdAt: string;
  updatedAt: string;
}

export interface ExperimentQueueResult {
  experimentId: string;
  run: ExperimentRun;
}

export interface SchedulerDecision {
  id: string;
  runId: string;
  chosenServerId?: string;
  decisionJson: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface SchedulerCandidate {
  serverId: string;
  serverName: string;
  heartbeatAt: string;
  status: string;
  bestGpuIndex: number;
  bestGpuName: string;
  bestFreeMemMb: number;
  bestUtilization: number;
  queueLength: number;
  snapshotCaptured?: string;
  eligible: boolean;
  ineligibleReason?: string;
}

export interface ScheduleResult {
  run: ExperimentRun;
  decision: SchedulerDecision;
  chosen: SchedulerCandidate;
}

export interface RunLog {
  id: number;
  runId: string;
  logType: string;
  logPath: string;
  tailText: string;
  createdAt: string;
  updatedAt: string;
}

export interface RunRecoveryDetail {
  runId: string;
  experimentId: string;
  runStatus: string;
  failureReason: string;
  failureStage: string;
  lastLogSummary: string;
  suggestRetry: boolean;
  retryCount: number;
  latestAssignedServerId: string;
}

export interface ResultComparison {
  id: string;
  experimentId: string;
  runId: string;
  baselineId?: string;
  targetResultArchiveId?: string;
  comparisonJson: Record<string, unknown>;
  summaryMd: string;
  createdAt: string;
  updatedAt: string;
}

export interface RunCompareResult {
  run: ExperimentRun;
  comparisons: ResultComparison[];
  resultArchive?: ResultArchiveDetail;
  workspaceDir: string;
  overallJudgment: string;
}

export interface ServerHeartbeat {
  id: string;
  serverId: string;
  heartbeatAt: string;
  status: string;
  detailJson: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface GPUResourceSnapshot {
  id: string;
  serverId: string;
  capturedAt: string;
  gpuIndex: number;
  name: string;
  totalMemMb: number;
  freeMemMb: number;
  utilization: number;
  processJson: Array<Record<string, unknown>>;
  createdAt: string;
  updatedAt: string;
}

export type AgentJobStatus =
  | "registered"
  | "idle"
  | "waiting_input"
  | "ready"
  | "running"
  | "validating"
  | "repairing"
  | "succeeded"
  | "failed"
  | "paused"
  | string;

export interface AgentInputRef {
  ref_type: string;
  ref_id?: string;
  ref_path?: string;
  ref_version?: string;
  metadata?: Record<string, unknown>;
}

export interface AgentArtifactManifestItem {
  artifact_type: string;
  name: string;
  file_path: string;
  metadata?: Record<string, unknown>;
}

export interface AgentRepairAction {
  action: string;
  status: string;
  detail: string;
  metadata?: Record<string, unknown>;
}

export interface AgentToolUsage {
  tool_ref: string;
  status: string;
  summary: string;
  metadata?: Record<string, unknown>;
}

export interface AgentJob {
  id: string;
  agent_type: string;
  execution_mode: string;
  model_provider: string;
  model_name: string;
  prompt_version: string;
  input_refs: AgentInputRef[];
  output_schema_ref: string;
  skill_refs: string[];
  tool_refs: string[];
  memory_refs: string[];
  workspace_dir: string;
  metadata: Record<string, unknown>;
  trigger_event_id: string;
  dedup_key: string;
  retry_count: number;
  max_retries: number;
  concurrency_limit: number;
  status: AgentJobStatus;
  normalized_payload: Record<string, unknown>;
  artifact_manifest: AgentArtifactManifestItem[];
  repair_actions: AgentRepairAction[];
  tool_usages: AgentToolUsage[];
  warnings: string[];
  validation_status: string;
  repair_status: string;
  validation_errors: string[];
  error_message: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AgentArtifact {
  id: string;
  job_id: string;
  artifact_type: string;
  name: string;
  file_path: string;
  checksum: string;
  metadata_json: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface AgentEvent {
  id: string;
  event_type: string;
  source_ref: string;
  input_refs: AgentInputRef[];
  payload: Record<string, unknown>;
  status: string;
  triggered_job_ids: string[];
  error_message: string;
  created_at: string;
  processed_at?: string;
  updated_at: string;
}

export interface AgentSubscription {
  id: string;
  name: string;
  event_type: string;
  agent_type: string;
  enabled: boolean;
  execution_mode: string;
  model_provider: string;
  model_name: string;
  prompt_version: string;
  output_schema_ref: string;
  skill_refs: string[];
  tool_refs: string[];
  memory_refs: string[];
  trigger_rule: Record<string, unknown>;
  max_retries: number;
  concurrency_limit: number;
  created_at: string;
  updated_at: string;
}

export interface AgentSummary {
  agent_type: string;
  event_types: string[];
  execution_mode: string;
  model_provider: string;
  model_name: string;
  prompt_version: string;
  output_schema_ref: string;
  skill_refs: string[];
  tool_refs: string[];
  memory_refs: string[];
  concurrency_limit: number;
  max_retries: number;
  job_count: number;
  latest_job?: AgentJob;
  subscriptions: AgentSubscription[];
}

export interface ToolDefinition {
  tool_id: string;
  name: string;
  owner_agent_type: string;
  path: string;
  description: string;
  usage_md: string;
  input_schema: Record<string, unknown>;
  output_schema: Record<string, unknown>;
  test_status: string;
  version: string;
  created_at: string;
  updated_at: string;
}

export interface SkillDefinition {
  skill_id: string;
  name: string;
  description: string;
  skill_dir: string;
  entrypoint: string;
  dependencies: string[];
  created_at: string;
  updated_at: string;
}

export interface Phase4DatasetSplit {
  name: string;
  path?: string;
  sampleCount?: number;
  note?: string;
}

export interface Phase4DatasetProfile {
  id: string;
  datasetName: string;
  taskType: string;
  modalityComposition: string[];
  splits: Phase4DatasetSplit[];
  labelSchema: Record<string, unknown>;
  fileStructureSnapshot: Record<string, unknown>;
  sampleStatistics: Record<string, unknown>;
  officialMetric: string;
  officialBaseline: string;
  license: string;
  citation: string;
  knownDifficulties: string[];
  userNotes: string;
  metadata: Record<string, unknown>;
  sourceMode: string;
  serverId?: string;
  serverName?: string;
  serverPath: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4DatasetProfileCreateRequest {
  datasetName: string;
  taskType: string;
  modalityComposition?: string[];
  splits?: Phase4DatasetSplit[];
  labelSchema?: Record<string, unknown>;
  fileStructureSnapshot?: Record<string, unknown>;
  sampleStatistics?: Record<string, unknown>;
  officialMetric?: string;
  officialBaseline?: string;
  license?: string;
  citation?: string;
  knownDifficulties?: string[];
  userNotes?: string;
  metadata?: Record<string, unknown>;
  sourceMode?: string;
  serverId?: string;
  serverPath?: string;
  status?: string;
}

export interface Phase4ReaderManualPaperInput {
  title: string;
  abstract: string;
  sourceType: string;
  sourceUrl: string;
  openAccessUrl: string;
  venue: string;
  year: number;
  authors: string[];
  filePath: string;
  note: string;
}

export interface Phase4ReaderSource {
  id: string;
  datasetProfileId?: string;
  title: string;
  authors: string[];
  venue: string;
  publicationYear?: number;
  sourceType: string;
  sourceUrl: string;
  openAccessUrl?: string;
  qualityTier: string;
  rankingScore: number;
  qualityScore: number;
  relevanceScore: number;
  citationCount: number;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4ReaderContext {
  id: string;
  datasetProfileId?: string;
  title: string;
  summary: string;
  taskDefinition: string;
  relatedWork: string[];
  retrievalFocus: string[];
  rankingNotes: string;
  sourceIds: string[];
  structuredContext: Record<string, unknown>;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4IdeaScore {
  novelty: number;
  datasetFit: number;
  feasibility: number;
  expectedGain: number;
  computeCost: number;
  failureRisk: number;
  reproducibility: number;
}

export interface Phase4Idea {
  id: string;
  datasetProfileId?: string;
  readerContextId?: string;
  title: string;
  problemDefinition: string;
  coreMethod: string;
  differentiators: string;
  dataProcessingNeeds: string[];
  modelChanges: string[];
  trainingPlan: string;
  evaluationMetrics: string[];
  riskPoints: string[];
  expectedGains: string[];
  score: Phase4IdeaScore;
  scoreSummary: Record<string, unknown>;
  status: string;
  sourceType: string;
  revisionOfId?: string;
  lineageRootId?: string;
  failureFeedback: Record<string, unknown>;
  lastFailureRunId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4IdeaScoreView {
  id: string;
  datasetProfileId?: string;
  readerContextId?: string;
  title: string;
  status: string;
  sourceType: string;
  revisionOfId?: string;
  lineageRootId?: string;
  lastFailureRunId?: string;
  score: Phase4IdeaScore;
  overallScore: number;
  rank: number;
  recommendationTier: string;
  recommendationReason: string;
  expectedGains: string[];
  riskPoints: string[];
}

export interface Phase4RunManifest {
  id: string;
  datasetProfileId: string;
  ideaId: string;
  readerContextId?: string;
  codeSnapshotId?: string;
  runnerMode: string;
  serverId?: string;
  gpu?: string;
  status: string;
  retryCount: number;
  maxRetryCount: number;
  artifactPaths: Record<string, unknown>;
  logsPath?: string;
  metricsPath?: string;
  failureFeedback: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  startedAt?: string;
  finishedAt?: string;
}

export interface Phase4StructuredReportRecord {
  id: string;
  runManifestId: string;
  datasetProfileId?: string;
  ideaId?: string;
  readerContextId?: string;
  title: string;
  machineReadableReport: Record<string, unknown>;
  humanReadableReportMd: string;
  citationRefs: string[];
  referenceSourceIds: string[];
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4WorkflowReaderConfig {
  manualPapers?: Phase4ReaderManualPaperInput[];
  userNotes?: string;
  searchMode?: string;
  maxPapers?: number;
  executionMode?: string;
  modelProvider?: string;
  modelName?: string;
  promptVersion?: string;
  skillRefs?: string[];
  toolRefs?: string[];
  memoryRefs?: string[];
}

export interface Phase4WorkflowIdeaConfig {
  userNotes?: string;
  targetCount?: number;
  executionMode?: string;
  modelProvider?: string;
  modelName?: string;
  promptVersion?: string;
  skillRefs?: string[];
  toolRefs?: string[];
  memoryRefs?: string[];
}

export interface Phase4WorkflowCodingConfig {
  runnerMode?: string;
  serverId?: string;
  gpu?: string;
  maxRetryCount?: number;
  userNotes?: string;
  executionMode?: string;
  modelProvider?: string;
  modelName?: string;
  promptVersion?: string;
  skillRefs?: string[];
  toolRefs?: string[];
  memoryRefs?: string[];
}

export interface Phase4WorkflowWritingConfig {
  userNotes?: string;
  executionMode?: string;
  modelProvider?: string;
  modelName?: string;
  promptVersion?: string;
  skillRefs?: string[];
  toolRefs?: string[];
  memoryRefs?: string[];
}

export interface Phase4Workflow {
  id: string;
  datasetProfileId: string;
  readerContextId?: string;
  selectedIdeaId?: string;
  currentRunManifestId?: string;
  latestReportId?: string;
  latestReaderJobId?: string;
  latestIdeaJobId?: string;
  latestCodingJobId?: string;
  latestWriterJobId?: string;
  status: string;
  nextAction: string;
  lastError: string;
  manualInputs: Record<string, unknown>;
  metadata: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4WorkflowAction {
  id: string;
  workflowId: string;
  stage: string;
  actionType: string;
  actorType: string;
  status: string;
  jobId?: string;
  runManifestId?: string;
  reportId?: string;
  payload: Record<string, unknown>;
  errorMessage: string;
  createdAt: string;
  updatedAt: string;
}

export interface Phase4WorkflowNextAction {
  action: string;
  label: string;
  description: string;
}

export interface Phase4WorkflowLatestJobs {
  reader?: AgentJob;
  idea?: AgentJob;
  coding?: AgentJob;
  writer?: AgentJob;
}

export interface Phase4WorkflowDetail {
  workflow: Phase4Workflow;
  datasetProfile?: Phase4DatasetProfile;
  readerContext?: Phase4ReaderContext;
  ideas: Phase4Idea[];
  topRecommendations: Phase4IdeaScoreView[];
  selectedIdea?: Phase4Idea;
  currentRunManifest?: Phase4RunManifest;
  latestReport?: Phase4StructuredReportRecord;
  latestJobs: Phase4WorkflowLatestJobs;
  nextActions: Phase4WorkflowNextAction[];
  timeline: Phase4WorkflowAction[];
}

export interface Phase4WorkflowCreateRequest {
  datasetProfileId: string;
  reader?: Phase4WorkflowReaderConfig;
  idea?: Phase4WorkflowIdeaConfig;
  coding?: Phase4WorkflowCodingConfig;
  writing?: Phase4WorkflowWritingConfig;
  metadata?: Record<string, unknown>;
}

export interface Phase4WorkflowSelectIdeaRequest {
  ideaId: string;
  coding?: Phase4WorkflowCodingConfig;
  writing?: Phase4WorkflowWritingConfig;
  userNotes?: string;
}

export interface Phase4WorkflowRetryStageRequest {
  coding?: Phase4WorkflowCodingConfig;
  writing?: Phase4WorkflowWritingConfig;
  userNotes?: string;
}

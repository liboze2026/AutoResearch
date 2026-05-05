package datasetagent

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/model"
	assetservice "mrag-platform/backend/go/internal/service"
)

type memoryJobStore struct {
	items map[string]model.AgentJob
}

func newMemoryJobStore() *memoryJobStore { return &memoryJobStore{items: map[string]model.AgentJob{}} }

func (s *memoryJobStore) Create(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryJobStore) GetByID(_ context.Context, id string) (*model.AgentJob, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryJobStore) Update(_ context.Context, item model.AgentJob) error {
	s.items[item.ID] = item
	return nil
}

type memoryTriggerStore struct {
	items map[string]model.AgentJobTrigger
}

func newMemoryTriggerStore() *memoryTriggerStore {
	return &memoryTriggerStore{items: map[string]model.AgentJobTrigger{}}
}

func (s *memoryTriggerStore) Create(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryTriggerStore) Update(_ context.Context, item model.AgentJobTrigger) error {
	s.items[item.ID] = item
	return nil
}

type memoryArtifactStore struct {
	items map[string][]model.AgentArtifact
}

func newMemoryArtifactStore() *memoryArtifactStore {
	return &memoryArtifactStore{items: map[string][]model.AgentArtifact{}}
}

func (s *memoryArtifactStore) Create(_ context.Context, item model.AgentArtifact) error {
	s.items[item.JobID] = append(s.items[item.JobID], item)
	return nil
}

type memoryDatasetAssetStore struct {
	assets          map[string]model.DatasetAsset
	sources         map[string][]model.DatasetAssetSource
	byDatasetSource map[string]string
}

func newMemoryDatasetAssetStore() *memoryDatasetAssetStore {
	return &memoryDatasetAssetStore{
		assets:          map[string]model.DatasetAsset{},
		sources:         map[string][]model.DatasetAssetSource{},
		byDatasetSource: map[string]string{},
	}
}

func (s *memoryDatasetAssetStore) List(_ context.Context) ([]model.DatasetAsset, error) {
	items := make([]model.DatasetAsset, 0, len(s.assets))
	for _, item := range s.assets {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryDatasetAssetStore) GetByID(_ context.Context, id string) (*model.DatasetAsset, error) {
	item, ok := s.assets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryDatasetAssetStore) GetByExistingDatasetRef(_ context.Context, datasetRef string) (*model.DatasetAsset, error) {
	assetID, ok := s.byDatasetSource[datasetRef]
	if !ok {
		return nil, nil
	}
	item := s.assets[assetID]
	copyItem := item
	return &copyItem, nil
}

func (s *memoryDatasetAssetStore) Create(_ context.Context, asset model.DatasetAsset) error {
	s.assets[asset.ID] = asset
	return nil
}

func (s *memoryDatasetAssetStore) Update(_ context.Context, asset model.DatasetAsset) error {
	s.assets[asset.ID] = asset
	return nil
}

func (s *memoryDatasetAssetStore) AddSource(_ context.Context, source model.DatasetAssetSource) error {
	s.sources[source.DatasetAssetID] = append(s.sources[source.DatasetAssetID], source)
	s.byDatasetSource[source.ExistingDatasetRef] = source.DatasetAssetID
	return nil
}

func (s *memoryDatasetAssetStore) ListSources(_ context.Context, datasetAssetID string) ([]model.DatasetAssetSource, error) {
	items := s.sources[datasetAssetID]
	out := make([]model.DatasetAssetSource, len(items))
	copy(out, items)
	return out, nil
}

type memoryDatasetScanReader struct {
	datasets map[string]model.Dataset
}

func newMemoryDatasetScanReader() *memoryDatasetScanReader {
	return &memoryDatasetScanReader{datasets: map[string]model.Dataset{}}
}

func (s *memoryDatasetScanReader) GetSummaryByID(_ context.Context, id string) (*model.Dataset, error) {
	item, ok := s.datasets[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryDatasetScanReader) GetScanRecordByID(_ context.Context, id string) (*model.DatasetScanRecord, error) {
	return nil, nil
}

type memoryBaselineStore struct {
	items map[string]model.Baseline
}

func newMemoryBaselineStore() *memoryBaselineStore {
	return &memoryBaselineStore{items: map[string]model.Baseline{}}
}

func (s *memoryBaselineStore) List(_ context.Context) ([]model.Baseline, error) {
	items := make([]model.Baseline, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryBaselineStore) GetByID(_ context.Context, id string) (*model.Baseline, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	copyItem := item
	return &copyItem, nil
}

func (s *memoryBaselineStore) Create(_ context.Context, item model.Baseline) error {
	s.items[item.ID] = item
	return nil
}

func (s *memoryBaselineStore) Update(_ context.Context, item model.Baseline) error {
	s.items[item.ID] = item
	return nil
}

type fakeDatasetDiscovery struct {
	items []model.Dataset
}

func (f *fakeDatasetDiscovery) List(_ context.Context, keyword string, _ string, _ string) ([]model.Dataset, error) {
	if strings.TrimSpace(keyword) == "" {
		return append([]model.Dataset{}, f.items...), nil
	}
	matched := make([]model.Dataset, 0)
	for _, item := range f.items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(keyword)) {
			matched = append(matched, item)
		}
	}
	return matched, nil
}

type fakeServerInspector struct {
	servers map[string]model.Server
	gpus    map[string]*model.GPUProbeResult
}

func (f *fakeServerInspector) List(_ context.Context) ([]model.Server, error) {
	items := make([]model.Server, 0, len(f.servers))
	for _, item := range f.servers {
		items = append(items, item)
	}
	return items, nil
}

func (f *fakeServerInspector) CheckGPU(_ context.Context, id string) (*model.GPUProbeResult, error) {
	item, ok := f.gpus[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

type fakeMemoryWriter struct {
	items map[string]model.AgentMemoryRecord
}

func newFakeMemoryWriter() *fakeMemoryWriter {
	return &fakeMemoryWriter{items: map[string]model.AgentMemoryRecord{}}
}

func (f *fakeMemoryWriter) Upsert(_ context.Context, req model.AgentMemoryUpsertRequest) (*model.AgentMemoryRecord, error) {
	item := model.AgentMemoryRecord{
		ID:        req.AgentType + "_" + req.MemoryKey,
		AgentType: req.AgentType,
		MemoryKey: req.MemoryKey,
		ContentMD: req.ContentMD,
		SourceRef: req.SourceRef,
		UpdatedAt: time.Now(),
		CreatedAt: time.Now(),
	}
	f.items[item.AgentType+"::"+item.MemoryKey] = item
	return &item, nil
}

func requirePython(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python not available")
	}
	return python
}

func pythonAgentsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "python_agents"))
}

func setupDatasetServices(t *testing.T, discovered []model.Dataset, servers []model.Server, gpus map[string]*model.GPUProbeResult) (*Service, string) {
	t.Helper()
	workspaceRoot := t.TempDir()
	jobStore := newMemoryJobStore()
	triggerStore := newMemoryTriggerStore()
	artifactStore := newMemoryArtifactStore()
	assetStore := newMemoryDatasetAssetStore()
	scanReader := newMemoryDatasetScanReader()
	for _, item := range discovered {
		scanReader.datasets[item.ID] = item
	}
	baselineStore := newMemoryBaselineStore()
	datasetAssetSvc := assetservice.NewDatasetAssetService(assetStore, scanReader, workspaceRoot)
	baselineSvc := assetservice.NewBaselineService(baselineStore, assetStore, workspaceRoot)
	datasetDiscovery := &fakeDatasetDiscovery{items: discovered}
	serverMap := map[string]model.Server{}
	for _, item := range servers {
		serverMap[item.ID] = item
	}
	serverSvc := &fakeServerInspector{servers: serverMap, gpus: gpus}
	memorySvc := newFakeMemoryWriter()

	pythonExec := requirePython(t)
	pythonDir := pythonAgentsDir(t)
	jobSvc := agentjob.NewService(jobStore, workspaceRoot)
	runtimeSvc := agentruntime.NewService(pythonExec, pythonDir, workspaceRoot)
	triggerSvc := agenttrigger.NewService(jobStore, triggerStore, artifactStore, runtimeSvc)
	datasetSvc := NewService(jobSvc, jobStore, triggerSvc, datasetAssetSvc, assetStore, baselineSvc, datasetDiscovery, serverSvc, memorySvc, workspaceRoot)
	triggerSvc.RegisterPostProcessor("dataset", datasetSvc)
	return datasetSvc, workspaceRoot
}

func TestDatasetServiceRunRegistersFromScan(t *testing.T) {
	discovered := []model.Dataset{
		{
			ID:               "ds_existing_1",
			Name:             "Retrieval Benchmark Suite",
			Description:      "Benchmark for retrieval experiments",
			Path:             "/datasets/retrieval-benchmark",
			ServerID:         "srv_shenzhen",
			ServerName:       "shenzhenvlab",
			Modality:         "text",
			DetectedModality: "retrieval",
			UpdatedAt:        time.Now(),
		},
	}
	servers := []model.Server{
		{ID: "srv_shenzhen", Name: "shenzhenvlab", Status: "online"},
		{ID: "srv_mock", Name: "mock_server", Status: "online"},
	}
	gpus := map[string]*model.GPUProbeResult{
		"srv_shenzhen": {
			ServerID:          "srv_shenzhen",
			ServerName:        "shenzhenvlab",
			AvailableGPUCount: 2,
			TotalGPUCount:     4,
			CheckedAt:         time.Now(),
		},
	}
	svc, workspaceRoot := setupDatasetServices(t, discovered, servers, gpus)

	result, err := svc.Run(context.Background(), model.DatasetRunRequest{
		ResearchDirection: "retrieval benchmark",
		TaskType:          "retrieval",
		Keywords:          []string{"retrieval", "benchmark"},
		ExecutionMode:     "mock",
	})
	if err != nil {
		t.Fatalf("dataset run failed: %v", err)
	}
	if result.DatasetAsset == nil {
		t.Fatalf("expected dataset asset")
	}
	if result.DatasetAsset.Asset.ExistingDatasetRef != "ds_existing_1" {
		t.Fatalf("expected registered existing dataset, got %s", result.DatasetAsset.Asset.ExistingDatasetRef)
	}
	if result.EvalPlan == nil || result.EvalPlan.EvalProtocolJSON["task_type"] != "retrieval" {
		t.Fatalf("expected retrieval eval plan")
	}
	if result.Baseline == nil {
		t.Fatalf("expected baseline placeholder to be created")
	}
	if _, err = os.Stat(filepath.Join(workspaceRoot, "datasets", result.DatasetAsset.Asset.ID, "evalplan.json")); err != nil {
		t.Fatalf("expected evalplan file: %v", err)
	}
}

func TestDatasetServiceRunMockDownloadFlow(t *testing.T) {
	servers := []model.Server{
		{ID: "srv_mock", Name: "mock_server", Status: "online"},
	}
	svc, workspaceRoot := setupDatasetServices(t, nil, servers, map[string]*model.GPUProbeResult{})

	result, err := svc.Run(context.Background(), model.DatasetRunRequest{
		ResearchDirection:      "few-shot reasoning",
		TaskType:               "classification",
		Keywords:               []string{"reasoning"},
		TargetServerPreference: "mock_server",
		ExecutionMode:          "mock",
	})
	if err != nil {
		t.Fatalf("dataset run failed: %v", err)
	}
	if result.DatasetAsset == nil {
		t.Fatalf("expected dataset asset")
	}
	if result.DatasetAsset.Asset.SourceType != "manual" {
		t.Fatalf("expected manual dataset asset source type, got %s", result.DatasetAsset.Asset.SourceType)
	}
	if !strings.Contains(result.DatasetAsset.Asset.LocalOrRemotePath, filepath.Join("downloads")) {
		t.Fatalf("expected mock download location, got %s", result.DatasetAsset.Asset.LocalOrRemotePath)
	}
	if _, err = os.Stat(filepath.Join(workspaceRoot, "datasets", result.DatasetAsset.Asset.ID, "evalplan.json")); err != nil {
		t.Fatalf("expected evalplan file: %v", err)
	}
}

func TestDatasetServiceFallsBackWhenShenzhenVlabUnavailable(t *testing.T) {
	servers := []model.Server{
		{ID: "srv_shenzhen", Name: "shenzhenvlab", Status: "online"},
		{ID: "srv_mock", Name: "mock_server", Status: "online"},
	}
	gpus := map[string]*model.GPUProbeResult{
		"srv_shenzhen": {
			ServerID:          "srv_shenzhen",
			ServerName:        "shenzhenvlab",
			AvailableGPUCount: 0,
			TotalGPUCount:     4,
			CheckedAt:         time.Now(),
		},
	}
	svc, _ := setupDatasetServices(t, nil, servers, gpus)

	result, err := svc.Run(context.Background(), model.DatasetRunRequest{
		ResearchDirection: "multimodal retrieval",
		TaskType:          "retrieval",
		Keywords:          []string{"retrieval"},
		ExecutionMode:     "mock",
	})
	if err != nil {
		t.Fatalf("dataset run failed: %v", err)
	}
	if result.EvalPlan == nil {
		t.Fatalf("expected eval plan")
	}
	if got := result.EvalPlan.ServerDecision["selected_server_name"]; got != "mock_server" {
		t.Fatalf("expected mock_server fallback, got %v", got)
	}
	foundFallbackWarning := false
	for _, item := range result.Warnings {
		if strings.Contains(strings.ToLower(item), "fell back to mock server") {
			foundFallbackWarning = true
			break
		}
	}
	if !foundFallbackWarning {
		t.Fatalf("expected fallback warning")
	}
}

func TestNormalizeFetchActionAcceptsRegisterExistingAlias(t *testing.T) {
	if got := normalizeFetchAction("register_existing_dataset"); got != "register_existing" {
		t.Fatalf("expected normalized register_existing, got %s", got)
	}
}

func TestShouldMaterializeDatasetLocationSkipsRemoteAliasPath(t *testing.T) {
	workspaceRoot := t.TempDir()
	location := "shenzhenvlab:/datasets/visdom"
	if shouldMaterializeDatasetLocation(location, "register_existing_dataset", workspaceRoot) {
		t.Fatalf("expected remote alias path %s to skip local mkdir", location)
	}
	if !isRemoteDatasetLocation(location) {
		t.Fatalf("expected remote alias path %s to be treated as remote", location)
	}
}

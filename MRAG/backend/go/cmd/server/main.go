package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"mrag-platform/backend/go/internal/agentadmin"
	"mrag-platform/backend/go/internal/agentartifact"
	"mrag-platform/backend/go/internal/agentjob"
	"mrag-platform/backend/go/internal/agentmemory"
	"mrag-platform/backend/go/internal/agentpipeline"
	"mrag-platform/backend/go/internal/agentruntime"
	"mrag-platform/backend/go/internal/agentschema"
	"mrag-platform/backend/go/internal/agenttrigger"
	"mrag-platform/backend/go/internal/codingagent"
	"mrag-platform/backend/go/internal/config"
	"mrag-platform/backend/go/internal/datasetagent"
	"mrag-platform/backend/go/internal/gpuresource"
	"mrag-platform/backend/go/internal/handler"
	"mrag-platform/backend/go/internal/heartbeat"
	"mrag-platform/backend/go/internal/ideaagent"
	"mrag-platform/backend/go/internal/insightagent"
	"mrag-platform/backend/go/internal/logcollector"
	"mrag-platform/backend/go/internal/model"
	"mrag-platform/backend/go/internal/phase4workflow"
	"mrag-platform/backend/go/internal/pkg/db"
	"mrag-platform/backend/go/internal/planneragent"
	"mrag-platform/backend/go/internal/readeragent"
	"mrag-platform/backend/go/internal/recovery"
	"mrag-platform/backend/go/internal/repository"
	"mrag-platform/backend/go/internal/resultcompare"
	"mrag-platform/backend/go/internal/router"
	"mrag-platform/backend/go/internal/runner"
	"mrag-platform/backend/go/internal/scheduler"
	"mrag-platform/backend/go/internal/service"
	"mrag-platform/backend/go/internal/skillregistry"
	"mrag-platform/backend/go/internal/toolregistry"
	"mrag-platform/backend/go/internal/traintemplate"
	"mrag-platform/backend/go/internal/writeragent"
)

func main() {
	cfg := config.Load()

	database, err := db.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer database.Close()

	migrationPath := filepath.Join("migrations")
	if err = db.RunMigrations(database, migrationPath); err != nil {
		log.Fatalf("run migration failed: %v", err)
	}

	datasetRepo := repository.NewDatasetRepository(database)
	datasetAssetRepo := repository.NewDatasetAssetRepository(database)
	baselineRepo := repository.NewBaselineRepository(database)
	resultArchiveRepo := repository.NewResultArchiveRepository(database)
	experimentRepo := repository.NewExperimentRepository(database)
	experimentSpecRepo := repository.NewExperimentSpecRepository(database)
	experimentRunRepo := repository.NewExperimentRunRepository(database)
	runLogRepo := repository.NewRunLogRepository(database)
	schedulerDecisionRepo := repository.NewSchedulerDecisionRepository(database)
	resultComparisonRepo := repository.NewResultComparisonRepository(database)
	serverRepo := repository.NewServerRepository(database)
	serverHeartbeatRepo := repository.NewServerHeartbeatRepository(database)
	gpuSnapshotRepo := repository.NewGPUResourceSnapshotRepository(database)
	overviewRepo := repository.NewOverviewRepository(database)
	paperRepo := repository.NewPaperRepository(database)
	ideaRepo := repository.NewIdeaRepository(database)
	phase4Repo := repository.NewPhase4Repository(database)
	agentJobRepo := repository.NewAgentJobRepository(database)
	agentArtifactRepo := repository.NewAgentArtifactRepository(database)
	agentTriggerRepo := repository.NewAgentJobTriggerRepository(database)
	agentSchemaRepo := repository.NewAgentSchemaRepository(database)
	agentEventRepo := repository.NewAgentEventRepository(database)
	agentSubscriptionRepo := repository.NewAgentSubscriptionRepository(database)
	toolRegistryRepo := repository.NewToolRegistryRepository(database)
	skillRegistryRepo := repository.NewSkillRegistryRepository(database)
	agentMemoryRepo := repository.NewAgentMemoryRepository(database)

	serverSSHGateway := service.NewSSHGateway(cfg.SSHBinary, cfg.SSHClientMode, cfg.SSHDialTimeoutSec)
	datasetRemoteSSHGateway := service.NewSSHGateway(cfg.SSHBinary, "real", cfg.SSHDialTimeoutSec)

	localScanRuntime := service.NewLocalDatasetRuntime(cfg.DatasetScanMode)
	remoteScanRuntime := service.NewRemoteDatasetRuntime(cfg.DatasetScanMode, datasetRemoteSSHGateway, cfg.RemoteDatasetRunnerEntrypoint, cfg.RemoteWorkRoot, cfg.SSHCommandTimeoutSec)
	localIndexRuntime := service.NewLocalDatasetRuntime(cfg.DatasetIndexMode)
	remoteIndexRuntime := service.NewRemoteDatasetRuntime(cfg.DatasetIndexMode, datasetRemoteSSHGateway, cfg.RemoteDatasetRunnerEntrypoint, cfg.RemoteWorkRoot, cfg.SSHCommandTimeoutSec)

	datasetScanResolver := service.NewDatasetScanAdapterResolver(
		service.NewDatasetScanAdapter(localScanRuntime),
		service.NewDatasetScanAdapter(remoteScanRuntime),
	)
	datasetIndexResolver := service.NewDatasetIndexAdapterResolver(
		service.NewDatasetIndexAdapter(localIndexRuntime),
		service.NewDatasetIndexAdapter(remoteIndexRuntime),
	)

	overviewAdapter := service.NewOverviewStatsAdapter(cfg.OverviewStatsMode, overviewRepo)
	runtimeProfileSvc := service.NewRuntimeProfileService(cfg)
	datasetSvc := service.NewDatasetService(datasetRepo, serverRepo, datasetScanResolver, datasetIndexResolver, cfg.DatasetPreviewLimit)
	datasetAssetSvc := service.NewDatasetAssetService(datasetAssetRepo, datasetRepo, cfg.WorkspaceRoot)
	baselineSvc := service.NewBaselineService(baselineRepo, datasetAssetRepo, cfg.WorkspaceRoot)
	resultArchiveSvc := service.NewResultArchiveService(resultArchiveRepo, datasetAssetRepo, ideaRepo, cfg.WorkspaceRoot)
	experimentSvc := service.NewExperimentService(experimentRepo, experimentSpecRepo, datasetAssetRepo, ideaRepo, baselineRepo, resultArchiveRepo, cfg.WorkspaceRoot)
	serverSvc := service.NewServerService(serverRepo, serverSSHGateway, cfg.SSHCommandTimeoutSec)
	schedulerSvc := scheduler.NewService(experimentRepo, experimentSpecRepo, experimentRunRepo, schedulerDecisionRepo, serverRepo, serverHeartbeatRepo, gpuSnapshotRepo, cfg.WorkspaceRoot)
	heartbeatSvc := heartbeat.NewService(serverSvc, serverHeartbeatRepo)
	gpuSnapshotSvc := gpuresource.NewService(serverSvc, gpuSnapshotRepo)
	templateSvc := traintemplate.NewService(cfg.PythonTemplatesDir, cfg.WorkspaceRoot)
	logCollectorSvc := logcollector.NewService(runLogRepo)
	recoverySvc := recovery.NewService(experimentRunRepo, experimentRepo, runLogRepo, cfg.WorkspaceRoot)
	resultCompareSvc := resultcompare.NewService(experimentRunRepo, experimentRepo, baselineRepo, resultArchiveRepo, resultComparisonRepo, resultArchiveSvc, cfg.WorkspaceRoot)
	runnerSvc := runner.NewService(experimentRunRepo, experimentRepo, experimentSpecRepo, serverRepo, runLogRepo, serverSSHGateway, templateSvc, resultCompareSvc, cfg.RemoteWorkRoot)
	overviewSvc := service.NewOverviewService(overviewAdapter, cfg.OverviewTrendDays)
	paperSvc := service.NewPaperService(paperRepo, cfg.PythonExec, cfg.PythonAgentsDir, cfg.WorkspaceRoot)
	ideaSvc := service.NewIdeaService(ideaRepo, paperRepo, cfg.WorkspaceRoot)
	phase4Svc := service.NewPhase4Service(phase4Repo)
	agentRuntimeSvc := agentruntime.NewService(cfg.PythonExec, cfg.PythonAgentsDir, cfg.WorkspaceRoot)
	agentJobSvc := agentjob.NewService(agentJobRepo, cfg.WorkspaceRoot)
	agentArtifactSvc := agentartifact.NewService(agentArtifactRepo)
	agentTriggerSvc := agenttrigger.NewService(agentJobRepo, agentTriggerRepo, agentArtifactRepo, agentRuntimeSvc)
	agentSchemaSvc := agentschema.NewService(agentSchemaRepo)
	agentPipelineSvc := agentpipeline.NewService(agentEventRepo, agentSubscriptionRepo, agentJobRepo, agentJobSvc, agentTriggerSvc)
	agentAdminSvc := agentadmin.NewService(agentJobRepo, agentArtifactRepo, agentEventRepo, agentSubscriptionRepo)
	toolRegistrySvc := toolregistry.NewService(toolRegistryRepo, cfg.WorkspaceRoot, cfg.PythonExec)
	skillRegistrySvc := skillregistry.NewService(skillRegistryRepo, cfg.WorkspaceRoot)
	agentMemorySvc := agentmemory.NewService(agentMemoryRepo, cfg.WorkspaceRoot)
	readerAgentSvc := readeragent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, agentArtifactSvc, paperSvc, cfg.WorkspaceRoot)
	phase4ReaderAgentSvc := readeragent.NewPhase4Service(agentJobSvc, agentJobRepo, agentTriggerSvc, agentArtifactSvc, phase4Svc, cfg.WorkspaceRoot)
	insightAgentSvc := insightagent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, paperSvc, cfg.WorkspaceRoot)
	datasetAgentSvc := datasetagent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, datasetAssetSvc, datasetAssetRepo, baselineSvc, datasetSvc, serverSvc, agentMemorySvc, cfg.WorkspaceRoot)
	ideaAgentSvc := ideaagent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, ideaSvc, paperSvc, datasetAssetSvc, cfg.WorkspaceRoot)
	phase4IdeaAgentSvc := ideaagent.NewPhase4Service(agentJobSvc, agentJobRepo, agentTriggerSvc, agentArtifactSvc, phase4Svc, cfg.WorkspaceRoot)
	phase4CodingAgentSvc := codingagent.NewPhase4Service(agentJobSvc, agentJobRepo, agentTriggerSvc, agentArtifactSvc, phase4Svc, agentPipelineSvc, cfg.WorkspaceRoot, cfg.PythonExec, cfg.PythonRunnersDir, cfg.Phase4RemoteWorkRoot)
	phase4CodingAgentSvc.ConfigureShenzhenlabExecution(serverRepo, serverSvc, serverSSHGateway, cfg.SSHCommandTimeoutSec)
	phase4CodingAgentSvc.AttachIdeaRevisionGenerator(phase4IdeaAgentSvc)
	phase4WriterAgentSvc := writeragent.NewPhase4Service(agentJobSvc, agentJobRepo, agentTriggerSvc, agentArtifactSvc, phase4Svc, cfg.WorkspaceRoot)
	phase4WorkflowSvc := phase4workflow.NewService(phase4Svc, phase4ReaderAgentSvc, phase4IdeaAgentSvc, phase4CodingAgentSvc, phase4WriterAgentSvc, agentJobSvc, agentPipelineSvc)
	plannerAgentSvc := planneragent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, experimentSvc, ideaSvc, datasetAssetSvc, baselineSvc, serverSvc, heartbeatSvc, gpuSnapshotSvc, agentPipelineSvc, cfg.WorkspaceRoot)
	codingAgentSvc := codingagent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, experimentSvc, ideaSvc, experimentSpecRepo, schedulerSvc, runnerSvc, resultCompareSvc, templateSvc, cfg.WorkspaceRoot, cfg.PythonTemplatesDir)
	writerAgentSvc := writeragent.NewService(agentJobSvc, agentJobRepo, agentTriggerSvc, ideaSvc, runnerSvc, resultCompareSvc, resultArchiveSvc, agentPipelineSvc, cfg.WorkspaceRoot)
	agentTriggerSvc.RegisterPostProcessor("insight", insightAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("dataset", datasetAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("idea_generator", ideaAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("planner", plannerAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("coding", codingAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("writer", writerAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("reader_phase4", phase4ReaderAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("idea_phase4", phase4IdeaAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("coding_phase4", phase4CodingAgentSvc)
	agentTriggerSvc.RegisterPostProcessor("writer_phase4", phase4WriterAgentSvc)
	ensureInsightSubscription(context.Background(), agentPipelineSvc, agentSubscriptionRepo)
	ensureIdeaSubscription(context.Background(), agentPipelineSvc, agentSubscriptionRepo)
	ensurePlannerSubscription(context.Background(), agentPipelineSvc, agentSubscriptionRepo)
	ensureCodingSubscription(context.Background(), agentPipelineSvc, agentSubscriptionRepo)
	agentpipeline.NewWorker(agentPipelineSvc, 2*time.Second, 20).Start(context.Background())
	heartbeat.NewMonitor(serverSvc, heartbeatSvc, gpuSnapshotSvc, cfg.ServerHeartbeatIntervalSec, cfg.GPUSnapshotIntervalSec).Start(context.Background())
	paperSvc.SetEventPublisher(agentPipelineSvc)
	datasetAssetSvc.SetEventPublisher(agentPipelineSvc)
	ideaSvc.SetEventPublisher(agentPipelineSvc)

	r := router.NewRouter(router.Dependencies{
		DatasetHandler:        handler.NewDatasetHandler(datasetSvc),
		DatasetAssetHandler:   handler.NewDatasetAssetHandler(datasetAssetSvc),
		BaselineHandler:       handler.NewBaselineHandler(baselineSvc),
		ResultArchiveHandler:  handler.NewResultArchiveHandler(resultArchiveSvc),
		ExperimentHandler:     handler.NewExperimentHandler(experimentSvc),
		ResultCompareHandler:  handler.NewResultCompareHandler(resultCompareSvc),
		SchedulerHandler:      handler.NewSchedulerHandler(schedulerSvc),
		RunHandler:            handler.NewRunHandler(runnerSvc, logCollectorSvc),
		RecoveryHandler:       handler.NewRecoveryHandler(recoverySvc),
		ServerHandler:         handler.NewServerHandler(serverSvc, heartbeatSvc, gpuSnapshotSvc),
		OverviewHandler:       handler.NewOverviewHandler(overviewSvc),
		RuntimeHandler:        handler.NewRuntimeHandler(runtimeProfileSvc),
		PaperHandler:          handler.NewPaperHandler(paperSvc),
		IdeaHandler:           handler.NewIdeaHandler(ideaSvc),
		IdeaAgentHandler:      handler.NewIdeaAgentHandler(ideaAgentSvc),
		PlannerAgentHandler:   handler.NewPlannerAgentHandler(plannerAgentSvc),
		CodingAgentHandler:    handler.NewCodingAgentHandler(codingAgentSvc),
		WriterAgentHandler:    handler.NewWriterAgentHandler(writerAgentSvc),
		ReaderAgentHandler:    handler.NewReaderAgentHandler(readerAgentSvc),
		InsightAgentHandler:   handler.NewInsightAgentHandler(insightAgentSvc),
		DatasetAgentHandler:   handler.NewDatasetAgentHandler(datasetAgentSvc),
		AgentJobHandler:       handler.NewAgentJobHandler(agentJobSvc, agentTriggerSvc, agentArtifactSvc),
		AgentAdminHandler:     handler.NewAgentAdminHandler(agentAdminSvc),
		AgentSchemaHandler:    handler.NewAgentSchemaHandler(agentSchemaSvc),
		ToolRegistryHandler:   handler.NewToolRegistryHandler(toolRegistrySvc),
		SkillRegistryHandler:  handler.NewSkillRegistryHandler(skillRegistrySvc),
		AgentMemoryHandler:    handler.NewAgentMemoryHandler(agentMemorySvc),
		Phase4Handler:         handler.NewPhase4Handler(phase4Svc),
		Phase4WorkflowHandler: handler.NewPhase4WorkflowHandler(phase4WorkflowSvc),
		Phase4ReaderHandler:   handler.NewPhase4ReaderHandler(phase4ReaderAgentSvc),
		Phase4IdeaHandler:     handler.NewPhase4IdeaHandler(phase4IdeaAgentSvc),
		Phase4CodingHandler:   handler.NewPhase4CodingHandler(phase4CodingAgentSvc),
		Phase4WriterHandler:   handler.NewPhase4WriterHandler(phase4WriterAgentSvc),
	})

	log.Printf("go backend started at :%s", cfg.Port)
	if err = r.Run(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

func ensureInsightSubscription(ctx context.Context, pipeline *agentpipeline.Service, repo *repository.AgentSubscriptionRepository) {
	if pipeline == nil || repo == nil {
		return
	}
	items, err := repo.ListByEventType(ctx, "paper_parsed")
	if err == nil {
		for _, item := range items {
			if item.AgentType == "insight" && item.Name == "stage3-default-insight-from-paper-parsed" {
				return
			}
		}
	}
	_, _ = pipeline.CreateSubscription(ctx, model.AgentSubscriptionCreateRequest{
		Name:            "stage3-default-insight-from-paper-parsed",
		EventType:       "paper_parsed",
		AgentType:       "insight",
		ExecutionMode:   "codex_cli",
		ModelProvider:   "codex",
		ModelName:       "insight-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/insight-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"parsed_content"},
		},
		MaxRetries: 2,
	})
}

func ensureIdeaSubscription(ctx context.Context, pipeline *agentpipeline.Service, repo *repository.AgentSubscriptionRepository) {
	if pipeline == nil || repo == nil {
		return
	}
	items, err := repo.ListByEventType(ctx, "insights_ready")
	if err == nil {
		for _, item := range items {
			if item.AgentType == "idea_generator" && item.Name == "stage3-default-idea-from-insights-ready" {
				return
			}
		}
	}
	_, _ = pipeline.CreateSubscription(ctx, model.AgentSubscriptionCreateRequest{
		Name:            "stage3-default-idea-from-insights-ready",
		EventType:       "insights_ready",
		AgentType:       "idea_generator",
		ExecutionMode:   "codex_cli",
		ModelProvider:   "codex",
		ModelName:       "idea-generator-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/idea-generator-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"insight"},
		},
		MaxRetries: 2,
	})
}

func ensurePlannerSubscription(ctx context.Context, pipeline *agentpipeline.Service, repo *repository.AgentSubscriptionRepository) {
	if pipeline == nil || repo == nil {
		return
	}
	items, err := repo.ListByEventType(ctx, "idea_ready")
	if err == nil {
		for _, item := range items {
			if item.AgentType == "planner" && item.Name == "stage3-default-planner-from-idea-ready" {
				return
			}
		}
	}
	_, _ = pipeline.CreateSubscription(ctx, model.AgentSubscriptionCreateRequest{
		Name:            "stage3-default-planner-from-idea-ready",
		EventType:       "idea_ready",
		AgentType:       "planner",
		ExecutionMode:   "codex_cli",
		ModelProvider:   "codex",
		ModelName:       "planner-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/planner-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"idea", "dataset_asset"},
		},
		MaxRetries: 2,
	})
}

func ensureCodingSubscription(ctx context.Context, pipeline *agentpipeline.Service, repo *repository.AgentSubscriptionRepository) {
	if pipeline == nil || repo == nil {
		return
	}
	items, err := repo.ListByEventType(ctx, "plan_ready")
	if err == nil {
		for _, item := range items {
			if item.AgentType == "coding" && item.Name == "stage3-default-coding-from-plan-ready" {
				return
			}
		}
	}
	_, _ = pipeline.CreateSubscription(ctx, model.AgentSubscriptionCreateRequest{
		Name:            "stage3-default-coding-from-plan-ready",
		EventType:       "plan_ready",
		AgentType:       "coding",
		ExecutionMode:   "codex_cli",
		ModelProvider:   "codex",
		ModelName:       "coding-default",
		PromptVersion:   "v1",
		OutputSchemaRef: "schemas/coding-output-v1.json",
		TriggerRule: map[string]any{
			"required_ref_types": []string{"experiment", "experiment_plan"},
		},
		MaxRetries: 1,
	})
}

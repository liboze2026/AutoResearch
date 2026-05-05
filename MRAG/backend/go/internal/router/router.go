package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"mrag-platform/backend/go/internal/handler"
)

type Dependencies struct {
	DatasetHandler        *handler.DatasetHandler
	DatasetAssetHandler   *handler.DatasetAssetHandler
	BaselineHandler       *handler.BaselineHandler
	ResultArchiveHandler  *handler.ResultArchiveHandler
	ExperimentHandler     *handler.ExperimentHandler
	ResultCompareHandler  *handler.ResultCompareHandler
	SchedulerHandler      *handler.SchedulerHandler
	RunHandler            *handler.RunHandler
	RecoveryHandler       *handler.RecoveryHandler
	ServerHandler         *handler.ServerHandler
	OverviewHandler       *handler.OverviewHandler
	RuntimeHandler        *handler.RuntimeHandler
	PaperHandler          *handler.PaperHandler
	IdeaHandler           *handler.IdeaHandler
	IdeaAgentHandler      *handler.IdeaAgentHandler
	PlannerAgentHandler   *handler.PlannerAgentHandler
	CodingAgentHandler    *handler.CodingAgentHandler
	WriterAgentHandler    *handler.WriterAgentHandler
	ReaderAgentHandler    *handler.ReaderAgentHandler
	InsightAgentHandler   *handler.InsightAgentHandler
	DatasetAgentHandler   *handler.DatasetAgentHandler
	AgentJobHandler       *handler.AgentJobHandler
	AgentAdminHandler     *handler.AgentAdminHandler
	AgentSchemaHandler    *handler.AgentSchemaHandler
	ToolRegistryHandler   *handler.ToolRegistryHandler
	SkillRegistryHandler  *handler.SkillRegistryHandler
	AgentMemoryHandler    *handler.AgentMemoryHandler
	Phase4Handler         *handler.Phase4Handler
	Phase4WorkflowHandler *handler.Phase4WorkflowHandler
	Phase4ReaderHandler   *handler.Phase4ReaderHandler
	Phase4IdeaHandler     *handler.Phase4IdeaHandler
	Phase4CodingHandler   *handler.Phase4CodingHandler
	Phase4WriterHandler   *handler.Phase4WriterHandler
}

func NewRouter(dep Dependencies) *gin.Engine {
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	apiV1 := r.Group("/api/v1")
	registerRoutes(apiV1, dep)

	apiV2 := r.Group("/api/v2")
	registerPhase4Routes(apiV2, dep)
	registerPhase4WorkflowRoutes(apiV2, dep)
	registerPhase4ReaderRoutes(apiV2, dep)
	registerPhase4IdeaRoutes(apiV2, dep)
	registerPhase4CodingRoutes(apiV2, dep)
	registerPhase4WriterRoutes(apiV2, dep)

	api := r.Group("/api")
	registerServerRoutes(api, dep)
	registerPaperRoutes(api, dep)
	registerReaderAgentRoutes(api, dep)
	registerInsightAgentRoutes(api, dep)
	registerDatasetAgentRoutes(api, dep)
	registerIdeaAgentRoutes(api, dep)
	registerPlannerAgentRoutes(api, dep)
	registerCodingAgentRoutes(api, dep)
	registerWriterAgentRoutes(api, dep)
	registerIdeaRoutes(api, dep)
	registerDatasetAssetRoutes(api, dep)
	registerBaselineRoutes(api, dep)
	registerResultArchiveRoutes(api, dep)
	registerExperimentRoutes(api, dep)
	registerSchedulerRoutes(api, dep)
	registerRunRoutes(api, dep)
	registerAgentAdminRoutes(api, dep)
	registerAgentJobRoutes(api, dep)
	registerAgentSchemaRoutes(api, dep)
	registerToolRegistryRoutes(api, dep)
	registerSkillRegistryRoutes(api, dep)
	registerAgentMemoryRoutes(api, dep)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}

func registerRoutes(api *gin.RouterGroup, dep Dependencies) {
	api.GET("/overview/stats", dep.OverviewHandler.Stats)
	api.GET("/runtime/profile", dep.RuntimeHandler.Profile)

	api.GET("/datasets", dep.DatasetHandler.List)
	api.GET("/datasets/:id", dep.DatasetHandler.Get)
	api.POST("/datasets/validate-path", dep.DatasetHandler.ValidatePath)
	api.POST("/datasets", dep.DatasetHandler.Create)
	api.PUT("/datasets/:id", dep.DatasetHandler.Update)
	api.DELETE("/datasets/:id", dep.DatasetHandler.Delete)
	api.POST("/datasets/:id/build-index", dep.DatasetHandler.BuildIndex)
	api.POST("/datasets/:id/index-tasks/:taskId/sync", dep.DatasetHandler.SyncIndexTask)

	registerServerRoutes(api, dep)

	registerPaperRoutes(api, dep)
	registerReaderAgentRoutes(api, dep)
	registerInsightAgentRoutes(api, dep)
	registerDatasetAgentRoutes(api, dep)
	registerIdeaAgentRoutes(api, dep)
	registerPlannerAgentRoutes(api, dep)
	registerCodingAgentRoutes(api, dep)
	registerWriterAgentRoutes(api, dep)
	registerIdeaRoutes(api, dep)
	registerDatasetAssetRoutes(api, dep)
	registerBaselineRoutes(api, dep)
	registerResultArchiveRoutes(api, dep)
	registerExperimentRoutes(api, dep)
	registerSchedulerRoutes(api, dep)
	registerRunRoutes(api, dep)
	registerAgentAdminRoutes(api, dep)
	registerAgentJobRoutes(api, dep)
	registerAgentSchemaRoutes(api, dep)
	registerToolRegistryRoutes(api, dep)
	registerSkillRegistryRoutes(api, dep)
	registerAgentMemoryRoutes(api, dep)
}

func registerServerRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.ServerHandler == nil {
		return
	}
	api.GET("/servers", dep.ServerHandler.List)
	api.POST("/servers", dep.ServerHandler.Create)
	api.PUT("/servers/:id", dep.ServerHandler.Update)
	api.DELETE("/servers/:id", dep.ServerHandler.Delete)
	api.POST("/servers/:id/test-connection", dep.ServerHandler.TestConnection)
	api.POST("/servers/:id/refresh-status", dep.ServerHandler.RefreshStatus)
	api.POST("/servers/:id/check-gpu", dep.ServerHandler.CheckGPU)
	api.POST("/servers/:id/heartbeat", dep.ServerHandler.Heartbeat)
	api.POST("/servers/:id/gpu-snapshot", dep.ServerHandler.GPUSnapshot)
	api.GET("/servers/:id/heartbeats", dep.ServerHandler.ListHeartbeats)
	api.GET("/servers/:id/gpu-snapshots", dep.ServerHandler.ListGPUSnapshots)
	api.POST("/servers/:id/scan-datasets", dep.ServerHandler.ScanDatasets)
}

func registerPaperRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.PaperHandler == nil {
		return
	}
	api.POST("/papers/import", dep.PaperHandler.Import)
	api.GET("/papers", dep.PaperHandler.List)
	api.GET("/papers/:id", dep.PaperHandler.Get)
	api.GET("/papers/:id/files", dep.PaperHandler.ListFiles)
	api.POST("/papers/:id/parse", dep.PaperHandler.Parse)
	api.POST("/papers/:id/extract-insights", dep.PaperHandler.ExtractInsights)
	api.GET("/papers/:id/insights", dep.PaperHandler.ListInsights)
}

func registerIdeaRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.IdeaHandler == nil {
		return
	}
	api.GET("/ideas", dep.IdeaHandler.List)
	api.POST("/ideas", dep.IdeaHandler.Create)
	api.GET("/ideas/:id", dep.IdeaHandler.Get)
	api.PATCH("/ideas/:id", dep.IdeaHandler.Update)
	api.POST("/ideas/generate-from-paper/:paperId", dep.IdeaHandler.GenerateFromPaper)
}

func registerReaderAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.ReaderAgentHandler == nil {
		return
	}
	api.POST("/agents/reader/run", dep.ReaderAgentHandler.Run)
	api.GET("/agents/reader/jobs/:id", dep.ReaderAgentHandler.GetJob)
}

func registerInsightAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.InsightAgentHandler == nil {
		return
	}
	api.POST("/agents/insight/run", dep.InsightAgentHandler.Run)
}

func registerDatasetAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.DatasetAgentHandler == nil {
		return
	}
	api.POST("/agents/dataset/run", dep.DatasetAgentHandler.Run)
	api.GET("/dataset-assets/:id/evalplan", dep.DatasetAgentHandler.GetEvalPlan)
}

func registerIdeaAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.IdeaAgentHandler == nil {
		return
	}
	api.POST("/agents/idea-generator/run", dep.IdeaAgentHandler.Run)
}

func registerPlannerAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.PlannerAgentHandler == nil {
		return
	}
	api.POST("/agents/planner/run", dep.PlannerAgentHandler.Run)
	api.GET("/experiments/:id/plan", dep.PlannerAgentHandler.GetPlan)
}

func registerCodingAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.CodingAgentHandler == nil {
		return
	}
	api.POST("/agents/coding/run", dep.CodingAgentHandler.Run)
	api.POST("/agents/evaluator/run", dep.CodingAgentHandler.RunEvaluator)
}

func registerWriterAgentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.WriterAgentHandler == nil {
		return
	}
	api.POST("/agents/writer/run", dep.WriterAgentHandler.Run)
	api.GET("/drafts/:id", dep.WriterAgentHandler.GetDraft)
}

func registerDatasetAssetRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.DatasetAssetHandler == nil {
		return
	}
	api.GET("/dataset-assets", dep.DatasetAssetHandler.List)
	api.POST("/dataset-assets/register-from-scan", dep.DatasetAssetHandler.RegisterFromScan)
	api.POST("/dataset-assets", dep.DatasetAssetHandler.Create)
	api.GET("/dataset-assets/:id", dep.DatasetAssetHandler.Get)
}

func registerBaselineRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.BaselineHandler == nil {
		return
	}
	api.GET("/baselines", dep.BaselineHandler.List)
	api.POST("/baselines", dep.BaselineHandler.Create)
	api.GET("/baselines/:id", dep.BaselineHandler.Get)
	api.PATCH("/baselines/:id", dep.BaselineHandler.Update)
}

func registerResultArchiveRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.ResultArchiveHandler == nil {
		return
	}
	api.GET("/result-archives", dep.ResultArchiveHandler.List)
	api.POST("/result-archives", dep.ResultArchiveHandler.Create)
	api.GET("/result-archives/:id", dep.ResultArchiveHandler.Get)
	api.PATCH("/result-archives/:id", dep.ResultArchiveHandler.Update)
}

func registerExperimentRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.ExperimentHandler == nil {
		return
	}
	api.POST("/experiments", dep.ExperimentHandler.Create)
	api.GET("/experiments", dep.ExperimentHandler.List)
	api.GET("/experiments/:id", dep.ExperimentHandler.Get)
	api.POST("/experiments/:id/generate-spec", dep.ExperimentHandler.GenerateSpec)
	api.GET("/experiments/:id/spec", dep.ExperimentHandler.GetSpec)
	if dep.ResultCompareHandler != nil {
		api.GET("/experiments/:id/comparisons", dep.ResultCompareHandler.ListByExperiment)
	}
}

func registerSchedulerRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.SchedulerHandler == nil {
		return
	}
	api.POST("/experiments/:id/queue", dep.SchedulerHandler.QueueExperiment)
	api.POST("/runs/:id/schedule", dep.SchedulerHandler.ScheduleRun)
	api.GET("/runs/:id/scheduler-decision", dep.SchedulerHandler.GetDecision)
}

func registerRunRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.RunHandler == nil {
		return
	}
	api.POST("/runs/:id/start", dep.RunHandler.Start)
	api.GET("/runs/:id", dep.RunHandler.Get)
	api.GET("/runs/:id/logs", dep.RunHandler.ListLogs)
	api.GET("/runs/:id/logs/tail", dep.RunHandler.TailLogs)
	if dep.ResultCompareHandler != nil {
		api.POST("/runs/:id/compare", dep.ResultCompareHandler.CompareRun)
	}
	if dep.RecoveryHandler != nil {
		api.POST("/runs/:id/retry", dep.RecoveryHandler.Retry)
		api.GET("/runs/:id/recovery", dep.RecoveryHandler.GetRecovery)
	}
}

func registerAgentJobRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.AgentJobHandler == nil {
		return
	}
	api.POST("/agent-jobs", dep.AgentJobHandler.Create)
	api.GET("/agent-jobs/:id", dep.AgentJobHandler.Get)
	api.GET("/agent-jobs/:id/status", dep.AgentJobHandler.GetStatus)
	api.POST("/agent-jobs/:id/trigger", dep.AgentJobHandler.Trigger)
	api.GET("/agent-jobs/:id/artifacts", dep.AgentJobHandler.ListArtifacts)
}

func registerAgentAdminRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.AgentAdminHandler == nil {
		return
	}
	api.GET("/agents", dep.AgentAdminHandler.ListAgents)
	api.GET("/agents/jobs", dep.AgentAdminHandler.ListJobs)
	api.GET("/agents/jobs/:id", dep.AgentAdminHandler.GetJob)
	api.GET("/agents/artifacts/:id", dep.AgentAdminHandler.ListArtifacts)
	api.GET("/agent-events", dep.AgentAdminHandler.ListEvents)
}

func registerAgentSchemaRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.AgentSchemaHandler == nil {
		return
	}
	api.GET("/agent-schemas", dep.AgentSchemaHandler.List)
	api.POST("/agent-schemas", dep.AgentSchemaHandler.Create)
	api.GET("/agent-schemas/:id", dep.AgentSchemaHandler.Get)
}

func registerToolRegistryRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.ToolRegistryHandler == nil {
		return
	}
	api.GET("/tools", dep.ToolRegistryHandler.List)
	api.POST("/tools/register", dep.ToolRegistryHandler.Register)
}

func registerSkillRegistryRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.SkillRegistryHandler == nil {
		return
	}
	api.GET("/skills", dep.SkillRegistryHandler.List)
	api.POST("/skills/register", dep.SkillRegistryHandler.Register)
}

func registerAgentMemoryRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.AgentMemoryHandler == nil {
		return
	}
	api.GET("/agents/:type/memory", dep.AgentMemoryHandler.ListByAgentType)
}

func registerPhase4Routes(api *gin.RouterGroup, dep Dependencies) {
	if dep.Phase4Handler == nil {
		return
	}
	api.GET("/phase4/dataset-profiles", dep.Phase4Handler.ListDatasetProfiles)
	api.POST("/phase4/dataset-profiles", dep.Phase4Handler.CreateDatasetProfile)
	api.GET("/phase4/dataset-profiles/:id", dep.Phase4Handler.GetDatasetProfile)
	api.PATCH("/phase4/dataset-profiles/:id", dep.Phase4Handler.UpdateDatasetProfile)
	api.DELETE("/phase4/dataset-profiles/:id", dep.Phase4Handler.DeleteDatasetProfile)

	api.GET("/phase4/reader-sources", dep.Phase4Handler.ListReaderSources)
	api.POST("/phase4/reader-sources", dep.Phase4Handler.CreateReaderSource)
	api.GET("/phase4/reader-sources/:id", dep.Phase4Handler.GetReaderSource)
	api.PATCH("/phase4/reader-sources/:id", dep.Phase4Handler.UpdateReaderSource)

	api.GET("/phase4/reader-contexts", dep.Phase4Handler.ListReaderContexts)
	api.POST("/phase4/reader-contexts", dep.Phase4Handler.CreateReaderContext)
	api.GET("/phase4/reader-contexts/:id", dep.Phase4Handler.GetReaderContext)
	api.PATCH("/phase4/reader-contexts/:id", dep.Phase4Handler.UpdateReaderContext)

	api.GET("/phase4/ideas", dep.Phase4Handler.ListIdeas)
	api.POST("/phase4/ideas", dep.Phase4Handler.CreateIdea)
	api.GET("/phase4/ideas/:id", dep.Phase4Handler.GetIdea)
	api.PATCH("/phase4/ideas/:id", dep.Phase4Handler.UpdateIdea)
	api.DELETE("/phase4/ideas/:id", dep.Phase4Handler.DeleteIdea)
	api.POST("/phase4/ideas/:id/status", dep.Phase4Handler.UpdateIdeaStatus)
	api.POST("/phase4/ideas/:id/select", dep.Phase4Handler.SelectIdea)
	api.POST("/phase4/ideas/:id/archive", dep.Phase4Handler.ArchiveIdea)
	api.POST("/phase4/ideas/:id/reject", dep.Phase4Handler.RejectIdea)
	api.GET("/phase4/ideas/score-view", dep.Phase4Handler.ListIdeaScoreViews)
	api.GET("/phase4/ideas/:id/score-view", dep.Phase4Handler.GetIdeaScoreView)

	api.GET("/phase4/runs", dep.Phase4Handler.ListRunManifests)
	api.POST("/phase4/runs", dep.Phase4Handler.CreateRunManifest)
	api.GET("/phase4/runs/:id", dep.Phase4Handler.GetRunManifest)
	api.PATCH("/phase4/runs/:id", dep.Phase4Handler.UpdateRunManifest)
	api.POST("/phase4/runs/:id/status", dep.Phase4Handler.UpdateRunManifestStatus)

	api.GET("/phase4/reports", dep.Phase4Handler.ListStructuredReports)
	api.POST("/phase4/reports", dep.Phase4Handler.CreateStructuredReport)
	api.GET("/phase4/reports/:id", dep.Phase4Handler.GetStructuredReport)
	api.PATCH("/phase4/reports/:id", dep.Phase4Handler.UpdateStructuredReport)
}

func registerPhase4ReaderRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.Phase4ReaderHandler == nil {
		return
	}
	api.POST("/phase4/reader/run", dep.Phase4ReaderHandler.Run)
	api.GET("/phase4/reader/jobs/:id", dep.Phase4ReaderHandler.GetJob)
}

func registerPhase4IdeaRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.Phase4IdeaHandler == nil {
		return
	}
	api.POST("/phase4/ideas/generate", dep.Phase4IdeaHandler.Run)
	api.POST("/phase4/ideas/:id/revisions/generate", dep.Phase4IdeaHandler.GenerateRevisionCandidates)
	api.GET("/phase4/ideas/jobs/:id", dep.Phase4IdeaHandler.GetJob)
}

func registerPhase4CodingRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.Phase4CodingHandler == nil {
		return
	}
	api.POST("/phase4/coding/run", dep.Phase4CodingHandler.Run)
	api.GET("/phase4/coding/jobs/:id", dep.Phase4CodingHandler.GetJob)
}

func registerPhase4WriterRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.Phase4WriterHandler == nil {
		return
	}
	api.POST("/phase4/writer/run", dep.Phase4WriterHandler.Run)
	api.GET("/phase4/writer/jobs/:id", dep.Phase4WriterHandler.GetJob)
}

func registerPhase4WorkflowRoutes(api *gin.RouterGroup, dep Dependencies) {
	if dep.Phase4WorkflowHandler == nil {
		return
	}
	api.POST("/phase4/workflows", dep.Phase4WorkflowHandler.Create)
	api.GET("/phase4/workflows", dep.Phase4WorkflowHandler.List)
	api.GET("/phase4/workflows/:id", dep.Phase4WorkflowHandler.Get)
	api.POST("/phase4/workflows/:id/select-idea", dep.Phase4WorkflowHandler.SelectIdea)
	api.POST("/phase4/workflows/:id/select-revision", dep.Phase4WorkflowHandler.SelectRevision)
	api.POST("/phase4/workflows/:id/retry-stage", dep.Phase4WorkflowHandler.RetryStage)
	api.POST("/phase4/workflows/:id/archive", dep.Phase4WorkflowHandler.Archive)
}

package workspace

import "path/filepath"

type Paths struct {
	Root string
}

func New(root string) Paths {
	return Paths{Root: root}
}

func (p Paths) PapersRoot() string {
	return filepath.Join(p.Root, "papers")
}

func (p Paths) PapersIncoming() string {
	return filepath.Join(p.PapersRoot(), "incoming")
}

func (p Paths) PapersParsed() string {
	return filepath.Join(p.PapersRoot(), "parsed")
}

func (p Paths) PapersInsights() string {
	return filepath.Join(p.PapersRoot(), "insights")
}

func (p Paths) IdeasRoot() string {
	return filepath.Join(p.Root, "ideas")
}

func (p Paths) IdeaPool() string {
	return filepath.Join(p.IdeasRoot(), "pool")
}

func (p Paths) WritingRoot() string {
	return filepath.Join(p.Root, "writing")
}

func (p Paths) DraftDir(draftID string) string {
	return filepath.Join(p.WritingRoot(), draftID)
}

func (p Paths) DatasetsRoot() string {
	return filepath.Join(p.Root, "datasets")
}

func (p Paths) DatasetAssetDir(assetID string) string {
	return filepath.Join(p.DatasetsRoot(), assetID)
}

func (p Paths) DatasetBaselinesDir(assetID string) string {
	return filepath.Join(p.DatasetAssetDir(assetID), "baselines")
}

func (p Paths) ResultsRoot() string {
	return filepath.Join(p.Root, "results")
}

func (p Paths) ExperimentsRoot() string {
	return filepath.Join(p.Root, "experiments")
}

func (p Paths) ExperimentDir(experimentID string) string {
	return filepath.Join(p.ExperimentsRoot(), experimentID)
}

func (p Paths) ExperimentComparisonsDir(experimentID string) string {
	return filepath.Join(p.ExperimentDir(experimentID), "comparisons")
}

func (p Paths) ResultArchiveDir(archiveID string) string {
	return filepath.Join(p.ResultsRoot(), archiveID)
}

func (p Paths) Phase4Root() string {
	return filepath.Join(p.Root, "phase4")
}

func (p Paths) Phase4RunsRoot() string {
	return filepath.Join(p.Phase4Root(), "runs")
}

func (p Paths) Phase4RunDir(runID string) string {
	return filepath.Join(p.Phase4RunsRoot(), runID)
}

func (p Paths) Phase4ArtifactsRoot() string {
	return filepath.Join(p.Phase4Root(), "artifacts")
}

func (p Paths) Phase4ArtifactDir(runID string) string {
	return filepath.Join(p.Phase4ArtifactsRoot(), runID)
}

func (p Paths) Phase4CacheDir() string {
	return filepath.Join(p.Phase4Root(), "cache")
}

func (p Paths) Phase4EnvsDir() string {
	return filepath.Join(p.Phase4Root(), "envs")
}

func (p Paths) MemoryRoot() string {
	return filepath.Join(p.Root, "memory")
}

func (p Paths) ToolsRoot() string {
	return filepath.Join(p.Root, "tools")
}

func (p Paths) GeneratedToolsRoot() string {
	return filepath.Join(p.ToolsRoot(), "generated")
}

func (p Paths) GeneratedToolDir(toolID string) string {
	return filepath.Join(p.GeneratedToolsRoot(), toolID)
}

func (p Paths) SkillsRoot() string {
	return filepath.Join(p.Root, "skills")
}

func (p Paths) SkillDir(skillID string) string {
	return filepath.Join(p.SkillsRoot(), skillID)
}

func (p Paths) AgentMemoryDir() string {
	return filepath.Join(p.MemoryRoot(), "agents")
}

func (p Paths) AgentMemoryTypeDir(agentType string) string {
	return filepath.Join(p.AgentMemoryDir(), agentType)
}

func (p Paths) AgentsRoot() string {
	return filepath.Join(p.Root, "agents")
}

func (p Paths) AgentJobsRoot() string {
	return filepath.Join(p.AgentsRoot(), "jobs")
}

func (p Paths) AgentJobDir(jobID string) string {
	return filepath.Join(p.AgentJobsRoot(), jobID)
}

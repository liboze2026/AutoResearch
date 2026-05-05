<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">{{ detail?.datasetProfile?.datasetName || "Phase4 Workflow" }}</h1>
        <p class="page-subtitle">Operate the full phase4 loop from one screen: dataset-driven Reader, manual idea selection, coding run observation, and report export.</p>
      </div>
      <el-space wrap>
        <el-button @click="load" :loading="loading">Refresh</el-button>
        <el-button v-if="canRetry" type="warning" :loading="retrying" @click="retryStage">Retry Failed Stage</el-button>
        <el-button v-if="detail?.latestReport" @click="exportMachineReport">Export JSON</el-button>
        <el-button v-if="detail?.latestReport" @click="exportHumanReport">Export MD</el-button>
        <el-button v-if="canArchive" :loading="archiving" @click="archiveWorkflow">Archive</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />
    <el-skeleton v-if="loading && !detail" :rows="10" animated />
    <el-empty v-else-if="!detail" description="Workflow was not found or is not loaded yet." />

    <template v-else>
      <el-row :gutter="12">
        <el-col :span="8">
          <el-card>
            <template #header>Workflow Overview</template>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Workflow ID">{{ detail.workflow.id }}</el-descriptions-item>
              <el-descriptions-item label="Status">{{ detail.workflow.status }}</el-descriptions-item>
              <el-descriptions-item label="Next Action">{{ detail.workflow.nextAction || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Last Error">{{ detail.workflow.lastError || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Updated">{{ formatDateTime(detail.workflow.updatedAt) }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card>
            <template #header>Dataset Profile</template>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Task">{{ detail.datasetProfile?.taskType || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Official Metric">{{ detail.datasetProfile?.officialMetric || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Server Path">{{ detail.datasetProfile?.serverPath || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Status">{{ detail.datasetProfile?.status || "-" }}</el-descriptions-item>
            </el-descriptions>
            <div class="section-space-sm">
              <el-space wrap>
                <el-tag v-for="item in detail.datasetProfile?.modalityComposition || []" :key="item">{{ item }}</el-tag>
              </el-space>
            </div>
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card>
            <template #header>Allowed Next Actions</template>
            <el-empty v-if="!detail.nextActions.length" description="No manual action is required right now." />
            <el-space v-else wrap>
              <el-tag v-for="item in detail.nextActions" :key="item.action" type="info">{{ item.label }}: {{ item.description }}</el-tag>
            </el-space>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>Reader Context</template>
            <el-empty v-if="!detail.readerContext" description="Reader has not produced a context yet." />
            <template v-else>
              <div class="section-title">{{ detail.readerContext.title }}</div>
              <p class="section-text">{{ detail.readerContext.summary || detail.readerContext.taskDefinition }}</p>
              <el-descriptions :column="1" border>
                <el-descriptions-item label="Task Definition">{{ detail.readerContext.taskDefinition || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Related Work">{{ detail.readerContext.relatedWork.join(", ") || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Retrieval Focus">{{ detail.readerContext.retrievalFocus.join(", ") || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Ranking Notes">{{ detail.readerContext.rankingNotes || "-" }}</el-descriptions-item>
              </el-descriptions>
              <div class="section-space-sm">
                <div class="section-title">Structured Context</div>
                <pre class="pre-block">{{ toPrettyJson(detail.readerContext.structuredContext) }}</pre>
              </div>
            </template>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>
              <div class="card-header">
                <span>Reader Sources and Citations</span>
                <el-button text @click="loadReaderSources" :loading="readerSourcesLoading">Refresh</el-button>
              </div>
            </template>
            <el-table :data="orderedReaderSources" size="small" empty-text="No reader source">
              <el-table-column prop="title" label="Paper" min-width="220" />
              <el-table-column label="Venue / Year" min-width="160">
                <template #default="{ row }">{{ row.venue || "-" }} {{ row.publicationYear || "" }}</template>
              </el-table-column>
              <el-table-column label="Tier" width="110">
                <template #default="{ row }">{{ row.qualityTier || row.sourceType }}</template>
              </el-table-column>
              <el-table-column label="Access" width="110">
                <template #default="{ row }">
                  <a v-if="row.openAccessUrl" :href="row.openAccessUrl" target="_blank" rel="noreferrer">Open</a>
                  <span v-else>-</span>
                </template>
              </el-table-column>
            </el-table>
            <div class="section-space-sm">
              <div class="section-title">Citation Metadata</div>
              <pre class="pre-block">{{ toPrettyJson(citationMetadata) }}</pre>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-card class="section-space">
        <template #header>Idea Pool</template>
        <div v-if="detail.selectedIdea" class="section-space-sm">
          <el-alert :title="`Selected idea: ${detail.selectedIdea.title}`" type="success" :closable="false" show-icon />
        </div>
        <el-row :gutter="12" class="section-space-sm">
          <el-col v-for="item in detail.topRecommendations" :key="item.id" :span="8">
            <el-card shadow="never" class="top-idea-card">
              <div class="section-title">{{ item.title }}</div>
              <div class="muted-text">Score: {{ Number(item.overallScore || 0).toFixed(2) }}</div>
              <div class="muted-text">Rank: {{ item.rank }}</div>
              <div class="muted-text">{{ item.recommendationReason }}</div>
            </el-card>
          </el-col>
        </el-row>
        <el-table :data="detail.ideas" size="small" empty-text="No phase4 idea">
          <el-table-column prop="title" label="Idea" min-width="220" />
          <el-table-column label="Score" width="100">
            <template #default="{ row }">{{ displayOverallScore(row) }}</template>
          </el-table-column>
          <el-table-column prop="status" label="Status" width="120" />
          <el-table-column label="Revision" min-width="150">
            <template #default="{ row }">{{ row.revisionOfId || "-" }}</template>
          </el-table-column>
          <el-table-column label="Recommended" width="120">
            <template #default="{ row }">
              <el-tag v-if="isTopRecommendation(row.id)" type="warning">Top</el-tag>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="Actions" width="180">
            <template #default="{ row }">
              <el-button
                v-if="detail.workflow.status === 'awaiting_selection'"
                text
                type="primary"
                :loading="selectingIdeaId === row.id"
                @click="selectIdea(row.id)"
              >
                Select
              </el-button>
              <el-button
                v-else-if="detail.workflow.status === 'awaiting_revision_selection' && isRevisionCandidate(row.id)"
                text
                type="primary"
                :loading="selectingIdeaId === row.id"
                @click="selectRevision(row.id)"
              >
                Select Revision
              </el-button>
              <el-button text @click="router.push('/ideas')">Open Pool</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>Coding Run</template>
            <el-empty v-if="!detail.currentRunManifest" description="Coding has not started yet." />
            <template v-else>
              <el-descriptions :column="1" border>
                <el-descriptions-item label="Run ID">{{ detail.currentRunManifest.id }}</el-descriptions-item>
                <el-descriptions-item label="Status">{{ detail.currentRunManifest.status }}</el-descriptions-item>
                <el-descriptions-item label="Runner">{{ detail.currentRunManifest.runnerMode }}</el-descriptions-item>
                <el-descriptions-item label="Server / GPU">
                  {{ detail.currentRunManifest.serverId || "-" }} / {{ detail.currentRunManifest.gpu || "-" }}
                </el-descriptions-item>
                <el-descriptions-item label="Retry">
                  {{ detail.currentRunManifest.retryCount }} / {{ detail.currentRunManifest.maxRetryCount }}
                </el-descriptions-item>
              </el-descriptions>

              <div class="subsection-title">Metrics</div>
              <el-table :data="metricRows" size="small" empty-text="Metrics will appear after evaluation completes.">
                <el-table-column prop="key" label="Metric" min-width="170" />
                <el-table-column prop="value" label="Value" min-width="120" />
              </el-table>

              <div class="subsection-title">Failure Summary</div>
              <pre class="pre-block">{{ toPrettyJson(errorSummary) }}</pre>

              <div class="subsection-title">Artifact Summary</div>
              <pre class="pre-block">{{ toPrettyJson(artifactSummary) }}</pre>

              <div class="subsection-title">Log Tails</div>
              <el-tabs>
                <el-tab-pane label="Driver Log">
                  <pre class="pre-block">{{ String(errorSummary.driver_log_tail || "-") }}</pre>
                </el-tab-pane>
                <el-tab-pane label="Run Log">
                  <pre class="pre-block">{{ String(errorSummary.run_log_tail || "-") }}</pre>
                </el-tab-pane>
              </el-tabs>

              <div v-if="codingJobArtifacts.length" class="subsection-title">Coding Job Artifacts</div>
              <el-table v-if="codingJobArtifacts.length" :data="codingJobArtifacts" size="small">
                <el-table-column prop="artifact_type" label="Type" width="150" />
                <el-table-column prop="name" label="Name" min-width="180" />
                <el-table-column prop="file_path" label="Path" min-width="260" show-overflow-tooltip />
              </el-table>
            </template>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>Writing Report</template>
            <el-empty v-if="!detail.latestReport" description="Writing report is not ready yet." />
            <template v-else>
              <div class="section-title">{{ detail.latestReport.title }}</div>
              <div class="section-space-sm">
                <el-space wrap>
                  <el-tag v-for="item in detail.latestReport.citationRefs" :key="item" type="info">{{ item }}</el-tag>
                </el-space>
              </div>

              <div class="subsection-title">Machine-readable Summary</div>
              <pre class="pre-block">{{ toPrettyJson(detail.latestReport.machineReadableReport) }}</pre>

              <div class="subsection-title">Human-readable Report</div>
              <pre class="pre-block">{{ detail.latestReport.humanReadableReportMd || "-" }}</pre>

              <div v-if="writerJobArtifacts.length" class="subsection-title">Writer Job Artifacts</div>
              <el-table v-if="writerJobArtifacts.length" :data="writerJobArtifacts" size="small">
                <el-table-column prop="artifact_type" label="Type" width="150" />
                <el-table-column prop="name" label="Name" min-width="180" />
                <el-table-column prop="file_path" label="Path" min-width="260" show-overflow-tooltip />
              </el-table>
            </template>
          </el-card>
        </el-col>
      </el-row>

      <el-card class="section-space">
        <template #header>Timeline</template>
        <el-table :data="detail.timeline" size="small" empty-text="No timeline item">
          <el-table-column prop="stage" label="Stage" width="120" />
          <el-table-column prop="actionType" label="Action" width="160" />
          <el-table-column prop="actorType" label="Actor" width="100" />
          <el-table-column prop="status" label="Status" width="110" />
          <el-table-column prop="jobId" label="Job" min-width="170" />
          <el-table-column prop="runManifestId" label="Run" min-width="170" />
          <el-table-column prop="reportId" label="Report" min-width="170" />
          <el-table-column label="Error" min-width="220">
            <template #default="{ row }">{{ row.errorMessage || "-" }}</template>
          </el-table-column>
          <el-table-column label="Created" width="170">
            <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
          </el-table-column>
        </el-table>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { phase4Api } from "@/api";
import type { AgentArtifact, Phase4Idea, Phase4StructuredReportRecord, Phase4WorkflowDetail } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { downloadTextFile, extractReportArtifactSummary, extractReportErrorSummary, extractReportMetrics, overallIdeaScore, readerCitationRows, toPrettyJson } from "@/views/phase4/phase4Ui";

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const retrying = ref(false);
const archiving = ref(false);
const selectingIdeaId = ref("");
const error = ref("");
const detail = ref<Phase4WorkflowDetail>();
const readerSources = ref<any[]>([]);
const readerSourcesLoading = ref(false);
const codingJobArtifacts = ref<AgentArtifact[]>([]);
const writerJobArtifacts = ref<AgentArtifact[]>([]);

const workflowId = computed(() => String(route.params.id || ""));
const topRecommendationIds = computed(() => new Set((detail.value?.topRecommendations || []).map((item) => item.id)));
const canRetry = computed(() => detail.value?.workflow.status === "blocked");
const canArchive = computed(() => !!detail.value && detail.value.workflow.status !== "archived");
const report = computed<Phase4StructuredReportRecord | undefined>(() => detail.value?.latestReport);
const metricRows = computed(() => extractReportMetrics(report.value));
const errorSummary = computed(() => extractReportErrorSummary(report.value));
const artifactSummary = computed(() => extractReportArtifactSummary(report.value));
const readerSourceBundle = computed(() => readerCitationRows(detail.value?.readerContext, readerSources.value));
const orderedReaderSources = computed(() => readerSourceBundle.value.orderedSources);
const citationMetadata = computed(() => readerSourceBundle.value.citationMetadata);

onMounted(async () => {
  await load();
});

async function load() {
  if (!workflowId.value) {
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    detail.value = await phase4Api.getPhase4WorkflowById(workflowId.value);
    await Promise.all([loadReaderSources(), loadJobArtifacts()]);
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

async function loadReaderSources() {
  if (!detail.value?.datasetProfile?.id) {
    readerSources.value = [];
    return;
  }
  readerSourcesLoading.value = true;
  try {
    readerSources.value = await phase4Api.getPhase4ReaderSources({ datasetProfileId: detail.value.datasetProfile.id });
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    readerSourcesLoading.value = false;
  }
}

async function loadJobArtifacts() {
  codingJobArtifacts.value = [];
  writerJobArtifacts.value = [];
  try {
    if (detail.value?.latestJobs.coding?.id) {
      const codingJob = await phase4Api.getPhase4CodingJob(detail.value.latestJobs.coding.id);
      codingJobArtifacts.value = codingJob.artifacts || [];
    }
    if (detail.value?.latestJobs.writer?.id) {
      const writerJob = await phase4Api.getPhase4WriterJob(detail.value.latestJobs.writer.id);
      writerJobArtifacts.value = writerJob.artifacts || [];
    }
  } catch (err) {
    ElMessage.error((err as Error).message);
  }
}

async function selectIdea(ideaId: string) {
  selectingIdeaId.value = ideaId;
  try {
    detail.value = await phase4Api.selectPhase4WorkflowIdea(workflowId.value, { ideaId });
    await Promise.all([loadReaderSources(), loadJobArtifacts()]);
    ElMessage.success("Idea selected and the workflow continued to coding/writing.");
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    selectingIdeaId.value = "";
  }
}

async function selectRevision(ideaId: string) {
  selectingIdeaId.value = ideaId;
  try {
    detail.value = await phase4Api.selectPhase4WorkflowRevision(workflowId.value, { ideaId });
    await Promise.all([loadReaderSources(), loadJobArtifacts()]);
    ElMessage.success("Revision idea selected and the workflow continued.");
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    selectingIdeaId.value = "";
  }
}

async function retryStage() {
  retrying.value = true;
  try {
    detail.value = await phase4Api.retryPhase4WorkflowStage(workflowId.value, {});
    await Promise.all([loadReaderSources(), loadJobArtifacts()]);
    ElMessage.success("Retried the blocked stage.");
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    retrying.value = false;
  }
}

async function archiveWorkflow() {
  archiving.value = true;
  try {
    detail.value = await phase4Api.archivePhase4Workflow(workflowId.value);
    ElMessage.success("Workflow archived.");
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    archiving.value = false;
  }
}

function exportMachineReport() {
  if (!detail.value?.latestReport) {
    return;
  }
  downloadTextFile(`${detail.value.latestReport.id}.json`, JSON.stringify(detail.value.latestReport.machineReadableReport || {}, null, 2), "application/json;charset=utf-8");
}

function exportHumanReport() {
  if (!detail.value?.latestReport) {
    return;
  }
  downloadTextFile(`${detail.value.latestReport.id}.md`, detail.value.latestReport.humanReadableReportMd || "");
}

function displayOverallScore(idea: Phase4Idea) {
  const score = Number(topRecommendationScore(idea.id) || overallIdeaScore(idea));
  return score ? score.toFixed(2) : "-";
}

function topRecommendationScore(ideaId: string) {
  return detail.value?.topRecommendations.find((item) => item.id === ideaId)?.overallScore || 0;
}

function isTopRecommendation(ideaId: string) {
  return topRecommendationIds.value.has(ideaId);
}

function isRevisionCandidate(ideaId: string) {
  return detail.value?.workflow.status === "awaiting_revision_selection" && topRecommendationIds.value.has(ideaId);
}
</script>

<style scoped>
.page-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.section-title {
  font-weight: 600;
  margin-bottom: 8px;
}

.section-text {
  color: var(--el-text-color-regular);
  margin-top: 0;
}

.pre-block {
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--panel-alt);
  border: 1px solid var(--border);
  padding: 12px;
  border-radius: 8px;
  max-height: 300px;
  overflow: auto;
}

.section-space-sm {
  margin-top: 12px;
}

.subsection-title {
  margin: 16px 0 10px;
  font-weight: 600;
}

.top-idea-card {
  min-height: 118px;
}

.muted-text {
  color: var(--el-text-color-secondary);
  margin-top: 6px;
}
</style>

<template>
  <div>
    <el-page-header @back="$router.push('/datasets')" content="Dataset Detail" />

    <div v-if="detail" class="section-space">
      <el-row :gutter="12">
        <el-col :span="16">
          <el-card>
            <template #header>
              <div class="card-header">
                <span>{{ detail.dataset.name }}</span>
                <StatusTag :status="detail.dataset.indexStatus" />
              </div>
            </template>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="Dataset ID">{{ detail.dataset.id }}</el-descriptions-item>
              <el-descriptions-item label="Storage">{{ detail.dataset.sourceType === "remote" ? "Remote path" : "Local path" }}</el-descriptions-item>
              <el-descriptions-item label="Server">{{ detail.dataset.serverName || detail.dataset.serverId || "Current backend" }}</el-descriptions-item>
              <el-descriptions-item label="Modality">{{ detail.dataset.detectedModality || detail.dataset.modality }}</el-descriptions-item>
              <el-descriptions-item label="Version">{{ detail.dataset.version }}</el-descriptions-item>
              <el-descriptions-item label="Last Scan Status">{{ detail.dataset.lastScanStatus || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Scale">{{ detail.dataset.fileCount }} files / {{ detail.dataset.directoryCount }} dirs</el-descriptions-item>
              <el-descriptions-item label="Size">{{ detail.dataset.size }}</el-descriptions-item>
              <el-descriptions-item label="Dataset Path" :span="2">{{ detail.dataset.path }}</el-descriptions-item>
              <el-descriptions-item label="Description" :span="2">{{ detail.dataset.description }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <el-card class="section-space">
            <template #header>Scan Summary</template>
            <template v-if="detail.latestScan">
              <el-row :gutter="12">
                <el-col :span="8"><div class="metric-card"><div class="metric-label">Files</div><div class="metric-value">{{ detail.latestScan.fileCount }}</div></div></el-col>
                <el-col :span="8"><div class="metric-card"><div class="metric-label">Directories</div><div class="metric-value">{{ detail.latestScan.directoryCount }}</div></div></el-col>
                <el-col :span="8"><div class="metric-card"><div class="metric-label">Total Size</div><div class="metric-value">{{ formatBytes(detail.latestScan.totalSizeBytes) }}</div></div></el-col>
              </el-row>
              <el-descriptions :column="2" border class="section-space">
                <el-descriptions-item label="Inferred Modality">{{ detail.latestScan.inferredModality }}</el-descriptions-item>
                <el-descriptions-item label="Runtime Mode">{{ detail.latestScan.runtimeMode }}</el-descriptions-item>
                <el-descriptions-item label="Recent Modified">{{ formatDateTime(detail.latestScan.recentModifiedAt) }}</el-descriptions-item>
                <el-descriptions-item label="Scanned At">{{ formatDateTime(detail.latestScan.scannedAt) }}</el-descriptions-item>
                <el-descriptions-item label="Root Path" :span="2">{{ detail.latestScan.rootPath }}</el-descriptions-item>
              </el-descriptions>

              <div class="subsection-title">File Type Distribution</div>
              <el-space wrap>
                <el-tag v-for="(count, type) in detail.latestScan.fileTypes" :key="type" size="small">{{ type }}: {{ count }}</el-tag>
              </el-space>

              <div class="subsection-title">Hierarchy Summary</div>
              <el-table :data="detail.latestScan.hierarchySummary" size="small" empty-text="No hierarchy summary">
                <el-table-column prop="level" label="Level" width="90" />
                <el-table-column prop="path" label="Path" min-width="220" />
                <el-table-column prop="itemCount" label="Items" width="100" />
              </el-table>
            </template>
            <el-empty v-else description="No scan record for this dataset yet." />
          </el-card>

          <el-card class="section-space">
            <template #header>Sample Preview</template>
            <el-table :data="detail.previewItems" size="small" empty-text="No preview items returned by the latest scan">
              <el-table-column prop="name" label="Name" min-width="180" />
              <el-table-column prop="itemType" label="Type" width="100" />
              <el-table-column prop="category" label="Category" width="120" />
              <el-table-column prop="relativePath" label="Relative Path" min-width="220" />
              <el-table-column label="Size" width="120">
                <template #default="scope">{{ scope.row.itemType === "directory" ? "-" : formatBytes(scope.row.sizeBytes) }}</template>
              </el-table-column>
            </el-table>
          </el-card>

          <el-card class="section-space">
            <template #header>
              <div class="card-header">
                <span>Phase4 Reader Results</span>
                <el-button text @click="loadPhase4Data" :loading="phase4Loading">Refresh</el-button>
              </div>
            </template>
            <el-empty v-if="!matchedProfile" description="Create a Phase4 Dataset Profile first to see Reader contexts and citations." />
            <template v-else>
              <el-alert
                v-if="latestReaderContext"
                class="section-space-sm"
                type="info"
                :closable="false"
                show-icon
                :title="latestReaderContext.title || 'Latest Reader Context'"
                :description="latestReaderContext.summary || latestReaderContext.taskDefinition"
              />
              <el-row :gutter="12">
                <el-col :span="12">
                  <div class="subsection-title">Reader Contexts</div>
                  <el-table :data="phase4Contexts" size="small" empty-text="No reader contexts">
                    <el-table-column prop="title" label="Title" min-width="180" />
                    <el-table-column prop="status" label="Status" width="110" />
                    <el-table-column label="Updated" width="170">
                      <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
                    </el-table-column>
                  </el-table>
                </el-col>
                <el-col :span="12">
                  <div class="subsection-title">Latest Structured Context</div>
                  <pre class="pre-block">{{ toPrettyJson(latestReaderContext?.structuredContext || {}) }}</pre>
                </el-col>
              </el-row>

              <div class="subsection-title">Reader Sources and Citations</div>
              <el-table :data="phase4Sources" size="small" empty-text="No reader sources">
                <el-table-column prop="title" label="Paper" min-width="240" />
                <el-table-column label="Venue / Year" min-width="160">
                  <template #default="{ row }">{{ row.venue || "-" }} {{ row.publicationYear || "" }}</template>
                </el-table-column>
                <el-table-column label="Tier" width="120">
                  <template #default="{ row }">{{ row.qualityTier || row.sourceType }}</template>
                </el-table-column>
                <el-table-column label="Open Access" width="120">
                  <template #default="{ row }">
                    <a v-if="row.openAccessUrl" :href="row.openAccessUrl" target="_blank" rel="noreferrer">Open</a>
                    <span v-else>-</span>
                  </template>
                </el-table-column>
                <el-table-column label="Citation Count" width="120">
                  <template #default="{ row }">{{ row.citationCount || 0 }}</template>
                </el-table-column>
              </el-table>
            </template>
          </el-card>
        </el-col>

        <el-col :span="8">
          <el-card>
            <template #header>
              <div class="card-header">
                <span>Phase4 Dataset Profile</span>
                <el-space>
                  <el-button size="small" @click="openProfileDialog">{{ matchedProfile ? "Edit" : "Create" }}</el-button>
                  <el-button
                    size="small"
                    type="primary"
                    :loading="workflowLoading"
                    :disabled="!detail.dataset.serverId && !matchedProfile"
                    @click="openPhase4Workflow"
                  >
                    Open Workflow
                  </el-button>
                </el-space>
              </div>
            </template>
            <el-skeleton v-if="phase4Loading" :rows="6" animated />
            <template v-else-if="matchedProfile">
              <el-descriptions :column="1" border>
                <el-descriptions-item label="Profile ID">{{ matchedProfile.id }}</el-descriptions-item>
                <el-descriptions-item label="Task">{{ matchedProfile.taskType }}</el-descriptions-item>
                <el-descriptions-item label="Official Metric">{{ matchedProfile.officialMetric || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Official Baseline">{{ matchedProfile.officialBaseline || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Server Path">{{ matchedProfile.serverPath }}</el-descriptions-item>
                <el-descriptions-item label="Status">{{ matchedProfile.status }}</el-descriptions-item>
                <el-descriptions-item label="Notes">{{ matchedProfile.userNotes || "-" }}</el-descriptions-item>
              </el-descriptions>
              <div class="subsection-title">Modality / Difficulties</div>
              <el-space wrap>
                <el-tag v-for="item in matchedProfile.modalityComposition" :key="item">{{ item }}</el-tag>
                <el-tag v-for="item in matchedProfile.knownDifficulties" :key="`risk-${item}`" type="warning">{{ item }}</el-tag>
              </el-space>
              <div class="subsection-title">Sample Statistics</div>
              <pre class="pre-block">{{ toPrettyJson(matchedProfile.sampleStatistics) }}</pre>
              <div class="subsection-title">File Structure Snapshot</div>
              <pre class="pre-block">{{ toPrettyJson(matchedProfile.fileStructureSnapshot) }}</pre>
              <div class="subsection-title">Linked Workflows</div>
              <el-table :data="phase4Workflows" size="small" empty-text="No workflows">
                <el-table-column prop="status" label="Status" width="170" />
                <el-table-column prop="nextAction" label="Next Action" min-width="120" />
                <el-table-column label="Open" width="90">
                  <template #default="{ row }">
                    <el-button text type="primary" @click="router.push(`/phase4/workflows/${row.id}`)">Open</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </template>
            <el-empty v-else description="No Phase4 Dataset Profile linked to this dataset yet." />
          </el-card>

          <el-card class="section-space">
            <template #header>Tags</template>
            <el-space wrap>
              <el-tag v-for="tag in detail.dataset.tags" :key="tag">{{ tag }}</el-tag>
            </el-space>
          </el-card>

          <el-card class="section-space">
            <template #header>
              <div class="card-header">
                <span>Index Tasks</span>
                <el-space>
                  <el-button size="small" @click="syncLatestTask" :disabled="!detail.latestIndexTask" :loading="syncing">Sync</el-button>
                  <el-button type="primary" size="small" @click="buildIndex" :loading="building">Build Index</el-button>
                </el-space>
              </div>
            </template>
            <div v-if="detail.latestIndexTask">
              <el-descriptions :column="1" border>
                <el-descriptions-item label="Task Status"><StatusTag :status="detail.latestIndexTask.status" /></el-descriptions-item>
                <el-descriptions-item label="Executor Mode">{{ detail.latestIndexTask.executorMode }}</el-descriptions-item>
                <el-descriptions-item label="Remote Task ID">{{ detail.latestIndexTask.remoteTaskId || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Log Path">{{ detail.latestIndexTask.logPath || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Status Path">{{ detail.latestIndexTask.statusPath || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Result Path">{{ detail.latestIndexTask.resultPath || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Updated">{{ formatDateTime(detail.latestIndexTask.updatedAt) }}</el-descriptions-item>
                <el-descriptions-item label="Error">{{ detail.latestIndexTask.errorMessage || "-" }}</el-descriptions-item>
              </el-descriptions>

              <div class="subsection-title">Task Logs</div>
              <el-timeline>
                <el-timeline-item v-for="log in detail.latestIndexTask.logs" :key="log.id" :timestamp="formatDateTime(log.createdAt)">
                  <strong>[{{ log.level }}]</strong> {{ log.content }}
                </el-timeline-item>
              </el-timeline>
            </div>
            <el-empty v-else description="No index task for this dataset yet." />
          </el-card>

          <el-card class="section-space">
            <template #header>History</template>
            <el-table :data="detail.indexTasks" size="small" empty-text="No historical index task">
              <el-table-column prop="id" label="Task ID" min-width="180" />
              <el-table-column label="Status" width="100">
                <template #default="scope"><StatusTag :status="scope.row.status" /></template>
              </el-table-column>
              <el-table-column label="Updated" min-width="150">
                <template #default="scope">{{ formatDateTime(scope.row.updatedAt) }}</template>
              </el-table-column>
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <el-empty v-else class="section-space" description="Dataset detail was not found." />

    <el-dialog v-model="profileDialogVisible" :title="matchedProfile ? 'Edit Phase4 Dataset Profile' : 'Create Phase4 Dataset Profile'" width="820px">
      <el-form label-width="150px">
        <el-form-item label="Dataset Name"><el-input v-model="profileForm.datasetName" /></el-form-item>
        <el-form-item label="Task Type"><el-input v-model="profileForm.taskType" /></el-form-item>
        <el-form-item label="Modality Composition">
          <el-input v-model="profileForm.modalityCompositionText" type="textarea" :rows="2" placeholder="One item per line or comma separated" />
        </el-form-item>
        <el-form-item label="Official Metric"><el-input v-model="profileForm.officialMetric" /></el-form-item>
        <el-form-item label="Official Baseline"><el-input v-model="profileForm.officialBaseline" /></el-form-item>
        <el-form-item label="License"><el-input v-model="profileForm.license" /></el-form-item>
        <el-form-item label="Citation"><el-input v-model="profileForm.citation" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="Known Difficulties">
          <el-input v-model="profileForm.knownDifficultiesText" type="textarea" :rows="3" placeholder="One item per line or comma separated" />
        </el-form-item>
        <el-form-item label="User Notes"><el-input v-model="profileForm.userNotes" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="Server Path"><el-input v-model="profileForm.serverPath" /></el-form-item>
        <el-form-item label="Splits JSON"><el-input v-model="profileForm.splitsJson" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="Label Schema JSON"><el-input v-model="profileForm.labelSchemaJson" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="File Snapshot JSON"><el-input v-model="profileForm.fileStructureSnapshotJson" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="Sample Statistics JSON"><el-input v-model="profileForm.sampleStatisticsJson" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="Metadata JSON"><el-input v-model="profileForm.metadataJson" type="textarea" :rows="4" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="profileDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="profileSubmitting" @click="submitProfile">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { datasetApi, phase4Api } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { DatasetDetail, Phase4DatasetProfile, Phase4Workflow } from "@/types/domain";
import { formatBytes, formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { downloadTextFile, joinLineList, matchPhase4DatasetProfile, parseJsonText, parseLineList, toPrettyJson } from "@/views/phase4/phase4Ui";

const route = useRoute();
const router = useRouter();

const detail = ref<DatasetDetail>();
const building = ref(false);
const syncing = ref(false);
const phase4Loading = ref(false);
const workflowLoading = ref(false);
const profileDialogVisible = ref(false);
const profileSubmitting = ref(false);
const phase4Profiles = ref<Phase4DatasetProfile[]>([]);
const phase4Contexts = ref<any[]>([]);
const phase4Sources = ref<any[]>([]);
const phase4Workflows = ref<Phase4Workflow[]>([]);

const profileForm = reactive({
  datasetName: "",
  taskType: "",
  modalityCompositionText: "",
  officialMetric: "",
  officialBaseline: "",
  license: "",
  citation: "",
  knownDifficultiesText: "",
  userNotes: "",
  serverPath: "",
  splitsJson: "[]",
  labelSchemaJson: "{}",
  fileStructureSnapshotJson: "{}",
  sampleStatisticsJson: "{}",
  metadataJson: "{}"
});

const matchedProfile = computed(() => matchPhase4DatasetProfile(phase4Profiles.value, detail.value?.dataset));
const latestReaderContext = computed(() => {
  return [...phase4Contexts.value].sort((left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime())[0];
});

onMounted(() => {
  void loadAll();
});

async function loadAll() {
  await loadDetail();
  await loadPhase4Data();
}

async function loadDetail() {
  try {
    detail.value = await datasetApi.getDatasetById(route.params.id as string);
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function loadPhase4Data() {
  if (!detail.value) {
    return;
  }
  phase4Loading.value = true;
  try {
    phase4Profiles.value = await phase4Api.getPhase4DatasetProfiles();
    const profile = matchPhase4DatasetProfile(phase4Profiles.value, detail.value.dataset);
    if (!profile) {
      phase4Contexts.value = [];
      phase4Sources.value = [];
      phase4Workflows.value = [];
      return;
    }
    const [contexts, sources, workflows] = await Promise.all([
      phase4Api.getPhase4ReaderContexts({ datasetProfileId: profile.id }),
      phase4Api.getPhase4ReaderSources({ datasetProfileId: profile.id }),
      phase4Api.getPhase4Workflows({ datasetProfileId: profile.id })
    ]);
    phase4Contexts.value = contexts;
    phase4Sources.value = sources;
    phase4Workflows.value = workflows;
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    phase4Loading.value = false;
  }
}

function openProfileDialog() {
  const profile = matchedProfile.value;
  const dataset = detail.value?.dataset;
  profileForm.datasetName = profile?.datasetName || dataset?.name || "";
  profileForm.taskType = profile?.taskType || "page_level_retrieval";
  profileForm.modalityCompositionText = joinLineList(profile?.modalityComposition || [dataset?.detectedModality || dataset?.modality || "multimodal"]);
  profileForm.officialMetric = profile?.officialMetric || "Recall@5";
  profileForm.officialBaseline = profile?.officialBaseline || "Phase4 initial retrieval baseline";
  profileForm.license = profile?.license || "";
  profileForm.citation = profile?.citation || "";
  profileForm.knownDifficultiesText = joinLineList(profile?.knownDifficulties || ["page-level retrieval first"]);
  profileForm.userNotes = profile?.userNotes || dataset?.description || "";
  profileForm.serverPath = profile?.serverPath || dataset?.path || "";
  profileForm.splitsJson = toPrettyJson(profile?.splits || []);
  profileForm.labelSchemaJson = toPrettyJson(profile?.labelSchema || {});
  profileForm.fileStructureSnapshotJson = toPrettyJson(
    profile?.fileStructureSnapshot || {
      sourceDatasetId: dataset?.id,
      path: dataset?.path
    }
  );
  profileForm.sampleStatisticsJson = toPrettyJson(
    profile?.sampleStatistics || {
      fileCount: dataset?.fileCount,
      directoryCount: dataset?.directoryCount,
      totalSizeBytes: dataset?.totalSizeBytes,
      detectedModality: dataset?.detectedModality || dataset?.modality
    }
  );
  profileForm.metadataJson = toPrettyJson(
    profile?.metadata || {
      sourceDatasetId: dataset?.id,
      sourceDatasetName: dataset?.name,
      tags: dataset?.tags
    }
  );
  profileDialogVisible.value = true;
}

async function submitProfile() {
  if (!detail.value) {
    return;
  }
  profileSubmitting.value = true;
  try {
    const payload = {
      datasetName: profileForm.datasetName.trim(),
      taskType: profileForm.taskType.trim(),
      modalityComposition: parseLineList(profileForm.modalityCompositionText),
      officialMetric: profileForm.officialMetric.trim(),
      officialBaseline: profileForm.officialBaseline.trim(),
      license: profileForm.license.trim(),
      citation: profileForm.citation.trim(),
      knownDifficulties: parseLineList(profileForm.knownDifficultiesText),
      userNotes: profileForm.userNotes.trim(),
      sourceMode: "registered_path",
      serverId: detail.value.dataset.serverId,
      serverPath: profileForm.serverPath.trim(),
      splits: parseJsonText(profileForm.splitsJson, []),
      labelSchema: parseJsonText(profileForm.labelSchemaJson, {}),
      fileStructureSnapshot: parseJsonText(profileForm.fileStructureSnapshotJson, {}),
      sampleStatistics: parseJsonText(profileForm.sampleStatisticsJson, {}),
      metadata: parseJsonText(profileForm.metadataJson, {})
    };
    if (matchedProfile.value) {
      await phase4Api.updatePhase4DatasetProfile(matchedProfile.value.id, payload);
      ElMessage.success("Phase4 Dataset Profile updated.");
    } else {
      await phase4Api.createPhase4DatasetProfile({
        ...payload,
        status: "active"
      });
      ElMessage.success("Phase4 Dataset Profile created.");
    }
    profileDialogVisible.value = false;
    await loadPhase4Data();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    profileSubmitting.value = false;
  }
}

async function openPhase4Workflow() {
  if (!detail.value) {
    return;
  }
  workflowLoading.value = true;
  try {
    let profile = matchedProfile.value;
    if (!profile) {
      openProfileDialog();
      ElMessage.info("Create a Phase4 Dataset Profile first.");
      return;
    }
    const workflows = await phase4Api.getPhase4Workflows({ datasetProfileId: profile.id });
    const activeWorkflow = workflows.find((item) => item.status !== "archived");
    if (activeWorkflow) {
      await router.push(`/phase4/workflows/${activeWorkflow.id}`);
      return;
    }
    await router.push(`/phase4/workflows?datasetProfileId=${profile.id}`);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    workflowLoading.value = false;
  }
}

async function buildIndex() {
  building.value = true;
  try {
    const task = await datasetApi.buildDatasetIndex(route.params.id as string);
    ElMessage.success(`Index task created: ${task.id}`);
    await loadDetail();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    building.value = false;
  }
}

async function syncLatestTask() {
  const latestTask = detail.value?.latestIndexTask;
  if (!latestTask) {
    ElMessage.warning("No index task to sync.");
    return;
  }
  syncing.value = true;
  try {
    await datasetApi.syncDatasetIndexTask(route.params.id as string, latestTask.id);
    ElMessage.success("Index task synced.");
    await loadDetail();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    syncing.value = false;
  }
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.metric-card {
  padding: 14px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
}

.metric-label {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.metric-value {
  font-size: 22px;
  font-weight: 600;
  margin-top: 8px;
}

.subsection-title {
  margin: 16px 0 10px;
  font-weight: 600;
}

.pre-block {
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--panel-alt);
  border: 1px solid var(--border);
  padding: 12px;
  border-radius: 8px;
  max-height: 260px;
  overflow: auto;
}
</style>

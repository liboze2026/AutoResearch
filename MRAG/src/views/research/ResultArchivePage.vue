<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Results and Reports</h1>
        <p class="page-subtitle">Reuse the existing result page for both stage3 result archives and phase4 structured experiment reports.</p>
      </div>
      <el-space wrap>
        <el-button @click="refreshActiveTab" :loading="loading || phase4Loading">Refresh</el-button>
        <el-button v-if="activeTab === 'stage3'" type="primary" @click="dialogVisible = true">Create Archive</el-button>
      </el-space>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="Stage3 Archives" name="stage3">
        <el-card>
          <el-table :data="archives" v-loading="loading" size="small" empty-text="No stage3 result archive">
            <el-table-column prop="title" label="Title" min-width="220" />
            <el-table-column prop="datasetAssetId" label="Dataset Asset" min-width="180" />
            <el-table-column prop="ideaId" label="Idea" min-width="180" />
            <el-table-column prop="status" label="Status" width="120" />
            <el-table-column label="Actions" width="180">
              <template #default="scope">
                <el-space>
                  <el-button text type="primary" @click="openDetail(scope.row.id)">Detail</el-button>
                  <el-button text @click="openEdit(scope.row.id)">Edit</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="Phase4 Reports" name="phase4">
        <el-card>
          <el-row :gutter="12" class="filter-row">
            <el-col :span="8">
              <el-select v-model="phase4Filters.datasetProfileId" placeholder="Dataset profile" clearable filterable style="width: 100%">
                <el-option v-for="item in phase4DatasetProfiles" :key="item.id" :label="item.datasetName" :value="item.id" />
              </el-select>
            </el-col>
            <el-col :span="8">
              <el-select v-model="phase4Filters.runManifestId" placeholder="Run manifest" clearable filterable style="width: 100%">
                <el-option v-for="item in phase4Runs" :key="item.id" :label="item.id" :value="item.id" />
              </el-select>
            </el-col>
            <el-col :span="8">
              <el-button type="primary" style="width: 100%" :loading="phase4Loading" @click="loadPhase4Reports">Apply Filters</el-button>
            </el-col>
          </el-row>

          <el-table :data="filteredPhase4Reports" v-loading="phase4Loading" size="small" empty-text="No phase4 report">
            <el-table-column prop="title" label="Title" min-width="260" />
            <el-table-column label="Dataset" min-width="180">
              <template #default="{ row }">{{ datasetProfileName(row.datasetProfileId) }}</template>
            </el-table-column>
            <el-table-column prop="runManifestId" label="Run" min-width="180" />
            <el-table-column prop="status" label="Status" width="120" />
            <el-table-column label="Updated" width="180">
              <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
            </el-table-column>
            <el-table-column label="Actions" width="220">
              <template #default="{ row }">
                <el-space wrap>
                  <el-button text type="primary" @click="openPhase4Report(row.id)">Detail</el-button>
                  <el-button text @click="exportPhase4ReportJson(row)">Export JSON</el-button>
                  <el-button text @click="exportPhase4ReportMd(row)">Export MD</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="detailVisible" title="Stage3 Archive Detail" size="48%">
      <el-skeleton v-if="detailLoading" :rows="8" animated />
      <template v-else-if="detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Title">{{ detail.archive.title }}</el-descriptions-item>
          <el-descriptions-item label="Dataset Asset">{{ detail.archive.datasetAssetId }}</el-descriptions-item>
          <el-descriptions-item label="Idea">{{ detail.archive.ideaId || "-" }}</el-descriptions-item>
          <el-descriptions-item label="Status">{{ detail.archive.status }}</el-descriptions-item>
          <el-descriptions-item label="Summary">{{ detail.archive.summaryMd }}</el-descriptions-item>
          <el-descriptions-item label="Note">{{ detail.archive.noteMd || "-" }}</el-descriptions-item>
        </el-descriptions>
        <div class="subsection-title">Metrics</div>
        <pre class="pre-block">{{ pretty(detail.archive.metricJson) }}</pre>
        <div class="subsection-title">Archive Files</div>
        <el-table :data="detail.files" size="small" empty-text="No archive file">
          <el-table-column prop="fileKind" label="Kind" width="120" />
          <el-table-column prop="filePath" label="Path" min-width="240" show-overflow-tooltip />
        </el-table>
      </template>
    </el-drawer>

    <el-drawer v-model="phase4DetailVisible" title="Phase4 Experiment Report" size="56%">
      <el-empty v-if="!phase4ReportDetail" description="No phase4 report selected" />
      <template v-else>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Title">{{ phase4ReportDetail.title }}</el-descriptions-item>
          <el-descriptions-item label="Dataset">{{ datasetProfileName(phase4ReportDetail.datasetProfileId) }}</el-descriptions-item>
          <el-descriptions-item label="Run Manifest">{{ phase4ReportDetail.runManifestId }}</el-descriptions-item>
          <el-descriptions-item label="Idea">{{ phase4ReportDetail.ideaId || "-" }}</el-descriptions-item>
          <el-descriptions-item label="Status">{{ phase4ReportDetail.status }}</el-descriptions-item>
        </el-descriptions>

        <div class="subsection-title">Machine-readable Summary</div>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Task">{{ phase4TaskDefinition }}</el-descriptions-item>
          <el-descriptions-item label="Primary Metric">{{ phase4PrimaryMetric }}</el-descriptions-item>
          <el-descriptions-item label="Citations">
            {{ phase4ReportDetail.citationRefs.join(", ") || "-" }}
          </el-descriptions-item>
        </el-descriptions>

        <div class="subsection-title">Metrics</div>
        <el-table :data="phase4MetricRows" size="small" empty-text="No parsed metrics">
          <el-table-column prop="key" label="Metric" min-width="180" />
          <el-table-column prop="value" label="Value" min-width="120" />
        </el-table>

        <div class="subsection-title">Error Summary</div>
        <pre class="pre-block">{{ pretty(phase4ErrorSummary) }}</pre>

        <div class="subsection-title">Human-readable Report</div>
        <pre class="pre-block">{{ phase4ReportDetail.humanReadableReportMd || "-" }}</pre>

        <div class="subsection-title">Machine-readable JSON</div>
        <pre class="pre-block">{{ pretty(phase4ReportDetail.machineReadableReport) }}</pre>

        <div class="subsection-title">Export</div>
        <el-space wrap>
          <el-button @click="exportPhase4ReportJson(phase4ReportDetail)">Export JSON</el-button>
          <el-button @click="exportPhase4ReportMd(phase4ReportDetail)">Export MD</el-button>
        </el-space>
      </template>
    </el-drawer>

    <el-dialog v-model="dialogVisible" :title="editingId ? 'Edit Archive' : 'Create Archive'" width="820px">
      <el-form label-width="130px">
        <el-form-item label="Title"><el-input v-model="form.title" /></el-form-item>
        <el-form-item label="Dataset Asset">
          <el-select v-model="form.datasetAssetId" filterable style="width: 100%">
            <el-option v-for="asset in assets" :key="asset.id" :label="asset.name" :value="asset.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Idea">
          <el-select v-model="form.ideaId" filterable clearable style="width: 100%">
            <el-option v-for="idea in ideas" :key="idea.id" :label="idea.title" :value="idea.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Status">
          <el-select v-model="form.status" style="width: 100%">
            <el-option label="archived" value="archived" />
            <el-option label="reviewed" value="reviewed" />
            <el-option label="draft" value="draft" />
          </el-select>
        </el-form-item>
        <el-form-item label="Summary"><el-input v-model="form.summaryMd" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="Metric JSON"><el-input v-model="metricText" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="Note"><el-input v-model="form.noteMd" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="Figure / Table Attachments">
          <el-input v-model="extraFigure" placeholder="Optional figure placeholder" class="section-inline" />
          <el-input v-model="extraTable" placeholder="Optional table placeholder" class="section-inline" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ideaApi, phase4Api, researchAssetApi } from "@/api";
import type { DatasetAsset, Idea, Phase4DatasetProfile, Phase4RunManifest, Phase4StructuredReportRecord, ResultArchive, ResultArchiveCreateRequest, ResultArchiveDetail } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { downloadTextFile, extractReportErrorSummary, extractReportMetrics } from "@/views/phase4/phase4Ui";

const activeTab = ref("stage3");

const archives = ref<ResultArchive[]>([]);
const assets = ref<DatasetAsset[]>([]);
const ideas = ref<Idea[]>([]);
const detail = ref<ResultArchiveDetail>();
const loading = ref(false);
const detailLoading = ref(false);
const detailVisible = ref(false);
const dialogVisible = ref(false);
const submitting = ref(false);
const editingId = ref("");
const metricText = ref('{\n  "accuracy": 0.88\n}');
const extraFigure = ref("");
const extraTable = ref("");

const phase4DatasetProfiles = ref<Phase4DatasetProfile[]>([]);
const phase4Runs = ref<Phase4RunManifest[]>([]);
const phase4Reports = ref<Phase4StructuredReportRecord[]>([]);
const phase4ReportDetail = ref<Phase4StructuredReportRecord>();
const phase4DetailVisible = ref(false);
const phase4Loading = ref(false);

const phase4Filters = reactive({
  datasetProfileId: "",
  runManifestId: ""
});

const phase4MetricRows = computed(() => extractReportMetrics(phase4ReportDetail.value));
const phase4ErrorSummary = computed(() => extractReportErrorSummary(phase4ReportDetail.value));
const phase4TaskDefinition = computed(() => {
  const task = phase4ReportDetail.value?.machineReadableReport?.task;
  if (task && typeof task === "object") {
    return String((task as Record<string, unknown>).definition || "-");
  }
  return "-";
});
const phase4PrimaryMetric = computed(() => {
  const metrics = phase4ReportDetail.value?.machineReadableReport?.metrics;
  if (metrics && typeof metrics === "object") {
    return String((metrics as Record<string, unknown>).primary_metric || "-");
  }
  return "-";
});
const filteredPhase4Reports = computed(() => {
  return phase4Reports.value.filter((item) => {
    if (phase4Filters.datasetProfileId && item.datasetProfileId !== phase4Filters.datasetProfileId) {
      return false;
    }
    if (phase4Filters.runManifestId && item.runManifestId !== phase4Filters.runManifestId) {
      return false;
    }
    return true;
  });
});

const form = reactive<ResultArchiveCreateRequest>({
  title: "",
  datasetAssetId: "",
  ideaId: "",
  summaryMd: "",
  status: "archived",
  noteMd: ""
});

onMounted(() => {
  void Promise.all([loadAll(), loadPhase4Reports()]);
});

watch(activeTab, async (value) => {
  if (value === "phase4" && !phase4Reports.value.length) {
    await loadPhase4Reports();
  }
});

async function refreshActiveTab() {
  if (activeTab.value === "phase4") {
    await loadPhase4Reports();
    return;
  }
  await loadAll();
}

async function loadAll() {
  loading.value = true;
  try {
    const [archiveList, assetList, ideaList] = await Promise.all([
      researchAssetApi.getResultArchives(),
      researchAssetApi.getDatasetAssets(),
      ideaApi.getIdeas()
    ]);
    archives.value = archiveList;
    assets.value = assetList;
    ideas.value = ideaList;
    if (!form.datasetAssetId) {
      form.datasetAssetId = assetList[0]?.id || "";
    }
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    loading.value = false;
  }
}

async function loadPhase4Reports() {
  phase4Loading.value = true;
  try {
    const [profiles, runs, reports] = await Promise.all([
      phase4Api.getPhase4DatasetProfiles(),
      phase4Api.getPhase4Runs({
        datasetProfileId: phase4Filters.datasetProfileId || undefined
      }),
      phase4Api.getPhase4Reports({
        runManifestId: phase4Filters.runManifestId || undefined
      })
    ]);
    phase4DatasetProfiles.value = profiles;
    phase4Runs.value = runs;
    phase4Reports.value = reports;
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    phase4Loading.value = false;
  }
}

async function openDetail(id: string) {
  detailVisible.value = true;
  detailLoading.value = true;
  try {
    detail.value = await researchAssetApi.getResultArchiveById(id);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    detailLoading.value = false;
  }
}

async function openPhase4Report(id: string) {
  phase4DetailVisible.value = true;
  try {
    phase4ReportDetail.value = await phase4Api.getPhase4ReportById(id);
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function openEdit(id: string) {
  const result = await researchAssetApi.getResultArchiveById(id);
  editingId.value = id;
  form.title = result.archive.title;
  form.datasetAssetId = result.archive.datasetAssetId;
  form.ideaId = result.archive.ideaId || "";
  form.summaryMd = result.archive.summaryMd;
  form.status = result.archive.status;
  form.noteMd = result.archive.noteMd;
  metricText.value = JSON.stringify(result.archive.metricJson || {}, null, 2);
  extraFigure.value = "";
  extraTable.value = "";
  dialogVisible.value = true;
}

async function submit() {
  submitting.value = true;
  try {
    const files = [] as { fileName: string; fileKind: string; content: string }[];
    if (extraFigure.value.trim()) {
      files.push({ fileName: "figure.txt", fileKind: "figure", content: extraFigure.value.trim() });
    }
    if (extraTable.value.trim()) {
      files.push({ fileName: "table.txt", fileKind: "table", content: extraTable.value.trim() });
    }
    const payload = {
      ...form,
      ideaId: form.ideaId || undefined,
      metricJson: JSON.parse(metricText.value || "{}"),
      files
    };
    if (editingId.value) {
      await researchAssetApi.updateResultArchive(editingId.value, payload);
      ElMessage.success("Stage3 archive updated.");
    } else {
      await researchAssetApi.createResultArchive(payload);
      ElMessage.success("Stage3 archive created.");
    }
    dialogVisible.value = false;
    resetForm();
    await loadAll();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
  }
}

function exportPhase4ReportJson(report: Phase4StructuredReportRecord) {
  downloadTextFile(`${report.id}.json`, JSON.stringify(report.machineReadableReport || {}, null, 2), "application/json;charset=utf-8");
}

function exportPhase4ReportMd(report: Phase4StructuredReportRecord) {
  downloadTextFile(`${report.id}.md`, report.humanReadableReportMd || "");
}

function datasetProfileName(datasetProfileId?: string) {
  return phase4DatasetProfiles.value.find((item) => item.id === datasetProfileId)?.datasetName || datasetProfileId || "-";
}

function pretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

function resetForm() {
  editingId.value = "";
  form.title = "";
  form.datasetAssetId = assets.value[0]?.id || "";
  form.ideaId = "";
  form.summaryMd = "";
  form.status = "archived";
  form.noteMd = "";
  metricText.value = '{\n  "accuracy": 0.88\n}';
  extraFigure.value = "";
  extraTable.value = "";
}
</script>

<style scoped>
.page-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
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
}

.section-inline + .section-inline {
  margin-top: 8px;
}

.filter-row {
  margin-bottom: 12px;
}
</style>

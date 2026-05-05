<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Idea Pool</h1>
        <p class="page-subtitle">Reuse the existing idea page for both stage3 ideas and the phase4 structured idea pool.</p>
      </div>
      <el-space wrap>
        <el-button @click="refreshActiveTab" :loading="loading || phase4Loading">Refresh</el-button>
        <template v-if="activeTab === 'stage3'">
          <el-button @click="generateDialogVisible = true">Generate from Paper</el-button>
          <el-button type="primary" @click="createDialogVisible = true">New Idea</el-button>
        </template>
      </el-space>
    </div>

    <el-tabs v-model="activeTab">
      <el-tab-pane label="Stage3 Ideas" name="stage3">
        <el-card>
          <el-table :data="ideas" v-loading="loading" size="small" empty-text="No stage3 idea">
            <el-table-column prop="title" label="Title" min-width="220" />
            <el-table-column prop="status" label="Status" width="120" />
            <el-table-column prop="sourceType" label="Source" width="120" />
            <el-table-column prop="priority" label="Priority" width="100" />
            <el-table-column prop="confidence" label="Confidence" width="110" />
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

      <el-tab-pane label="Phase4 Ideas" name="phase4">
        <el-card>
          <el-row :gutter="12" class="filter-row">
            <el-col :span="6">
              <el-select v-model="phase4Filters.datasetProfileId" placeholder="Dataset profile" clearable filterable style="width: 100%">
                <el-option v-for="item in phase4DatasetProfiles" :key="item.id" :label="item.datasetName" :value="item.id" />
              </el-select>
            </el-col>
            <el-col :span="4">
              <el-select v-model="phase4Filters.status" placeholder="Status" clearable style="width: 100%">
                <el-option v-for="item in phase4Statuses" :key="item" :label="item" :value="item" />
              </el-select>
            </el-col>
            <el-col :span="4">
              <el-select v-model="phase4Filters.sortBy" placeholder="Sort" style="width: 100%">
                <el-option label="Overall score" value="overallScore" />
                <el-option label="Updated time" value="updatedAt" />
                <el-option label="Title" value="title" />
                <el-option label="Status" value="status" />
              </el-select>
            </el-col>
            <el-col :span="4">
              <el-switch v-model="phase4Filters.revisionOnly" inline-prompt active-text="Revision" inactive-text="All" />
            </el-col>
            <el-col :span="6">
              <el-button type="primary" style="width: 100%" :loading="phase4Loading" @click="loadPhase4Ideas">Apply Filters</el-button>
            </el-col>
          </el-row>

          <el-table :data="filteredPhase4Ideas" v-loading="phase4Loading" size="small" empty-text="No phase4 idea">
            <el-table-column prop="title" label="Idea" min-width="240" />
            <el-table-column label="Dataset" min-width="160">
              <template #default="{ row }">{{ datasetProfileName(row.datasetProfileId) }}</template>
            </el-table-column>
            <el-table-column label="Score" width="100">
              <template #default="{ row }">{{ displayOverallScore(row) }}</template>
            </el-table-column>
            <el-table-column label="Rank" width="90">
              <template #default="{ row }">{{ scoreRank(row.id) || "-" }}</template>
            </el-table-column>
            <el-table-column prop="status" label="Status" width="120" />
            <el-table-column label="Revision" min-width="170">
              <template #default="{ row }">
                <span v-if="row.revisionOfId">{{ row.revisionOfId }}</span>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column label="Failure Run" min-width="150">
              <template #default="{ row }">{{ row.lastFailureRunId || "-" }}</template>
            </el-table-column>
            <el-table-column label="Actions" width="260">
              <template #default="{ row }">
                <el-space wrap>
                  <el-button text type="primary" @click="openPhase4Idea(row)">Detail</el-button>
                  <el-button text type="success" :loading="phase4ActionLoadingId === row.id" @click="selectPhase4Idea(row.id)">Select</el-button>
                  <el-button text type="warning" :loading="phase4ActionLoadingId === row.id" @click="rejectPhase4Idea(row.id)">Reject</el-button>
                  <el-button text :loading="phase4ActionLoadingId === row.id" @click="archivePhase4Idea(row.id)">Archive</el-button>
                  <el-button text type="danger" :loading="phase4ActionLoadingId === row.id" @click="deletePhase4Idea(row.id)">Delete</el-button>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="detailVisible" title="Stage3 Idea Detail" size="45%">
      <el-skeleton v-if="detailLoading" :rows="8" animated />
      <template v-else-if="detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Title">{{ detail.idea.title }}</el-descriptions-item>
          <el-descriptions-item label="Status">{{ detail.idea.status }}</el-descriptions-item>
          <el-descriptions-item label="Source">{{ detail.idea.sourceType }}</el-descriptions-item>
          <el-descriptions-item label="Description">{{ detail.idea.descriptionMd || "-" }}</el-descriptions-item>
        </el-descriptions>
        <div class="subsection-title">Sources</div>
        <el-table :data="detail.sources" size="small" empty-text="No source record">
          <el-table-column prop="paperTitle" label="Paper" min-width="180" />
          <el-table-column prop="sourceNote" label="Note" min-width="220" />
        </el-table>
      </template>
    </el-drawer>

    <el-drawer v-model="phase4DetailVisible" title="Phase4 Idea Detail" size="52%">
      <el-empty v-if="!phase4Detail" description="No phase4 idea selected" />
      <template v-else>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Title">{{ phase4Detail.title }}</el-descriptions-item>
          <el-descriptions-item label="Dataset">{{ datasetProfileName(phase4Detail.datasetProfileId) }}</el-descriptions-item>
          <el-descriptions-item label="Status">{{ phase4Detail.status }}</el-descriptions-item>
          <el-descriptions-item label="Source">{{ phase4Detail.sourceType }}</el-descriptions-item>
          <el-descriptions-item label="Problem Definition">{{ phase4Detail.problemDefinition }}</el-descriptions-item>
          <el-descriptions-item label="Core Method">{{ phase4Detail.coreMethod }}</el-descriptions-item>
          <el-descriptions-item label="Differentiators">{{ phase4Detail.differentiators || "-" }}</el-descriptions-item>
          <el-descriptions-item label="Training Plan">{{ phase4Detail.trainingPlan || "-" }}</el-descriptions-item>
          <el-descriptions-item label="Revision Of">{{ phase4Detail.revisionOfId || "-" }}</el-descriptions-item>
          <el-descriptions-item label="Lineage Root">{{ phase4Detail.lineageRootId || "-" }}</el-descriptions-item>
          <el-descriptions-item label="Last Failure Run">{{ phase4Detail.lastFailureRunId || "-" }}</el-descriptions-item>
        </el-descriptions>

        <div class="subsection-title">Engineering Fields</div>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>Data Processing</template>
              <el-space wrap>
                <el-tag v-for="item in phase4Detail.dataProcessingNeeds" :key="item">{{ item }}</el-tag>
              </el-space>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="never">
              <template #header>Model Changes</template>
              <el-space wrap>
                <el-tag v-for="item in phase4Detail.modelChanges" :key="item" type="success">{{ item }}</el-tag>
              </el-space>
            </el-card>
          </el-col>
        </el-row>

        <div class="subsection-title">Evaluation / Risks / Expected Gains</div>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-card shadow="never">
              <template #header>Metrics</template>
              <el-space wrap>
                <el-tag v-for="item in phase4Detail.evaluationMetrics" :key="item">{{ item }}</el-tag>
              </el-space>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never">
              <template #header>Risk Points</template>
              <el-space wrap>
                <el-tag v-for="item in phase4Detail.riskPoints" :key="item" type="warning">{{ item }}</el-tag>
              </el-space>
            </el-card>
          </el-col>
          <el-col :span="8">
            <el-card shadow="never">
              <template #header>Expected Gains</template>
              <el-space wrap>
                <el-tag v-for="item in phase4Detail.expectedGains" :key="item" type="success">{{ item }}</el-tag>
              </el-space>
            </el-card>
          </el-col>
        </el-row>

        <div class="subsection-title">Score Summary</div>
        <pre class="pre-block">{{ toPrettyJson(phase4Detail.scoreSummary) }}</pre>

        <div class="subsection-title">Failure Feedback</div>
        <pre class="pre-block">{{ toPrettyJson(phase4Detail.failureFeedback) }}</pre>
      </template>
    </el-drawer>

    <el-dialog v-model="createDialogVisible" title="New Stage3 Idea" width="680px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="Title"><el-input v-model="form.title" /></el-form-item>
        <el-form-item label="Description"><el-input v-model="form.descriptionMd" type="textarea" :rows="5" /></el-form-item>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="Status"><el-select v-model="form.status" style="width: 100%"><el-option label="draft" value="draft" /><el-option label="shortlisted" value="shortlisted" /><el-option label="archived" value="archived" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="Source"><el-select v-model="form.sourceType" style="width: 100%"><el-option label="human" value="human" /><el-option label="mixed" value="mixed" /><el-option label="auto" value="auto" /></el-select></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="Weight"><el-input-number v-model="form.weight" :min="0" :max="100" style="width: 100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="Priority"><el-input-number v-model="form.priority" :min="0" :max="100" style="width: 100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="Confidence"><el-input-number v-model="form.confidence" :min="0" :max="1" :step="0.01" style="width: 100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="Source Note"><el-input v-model="form.sourceNote" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">Save</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="generateDialogVisible" title="Generate Stage3 Idea from Paper" width="520px">
      <el-form label-width="100px">
        <el-form-item label="Paper">
          <el-select v-model="selectedPaperId" filterable placeholder="Select paper" style="width: 100%">
            <el-option v-for="paper in papers" :key="paper.id" :label="`${paper.title} (${paper.status})`" :value="paper.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="generateDialogVisible = false">Cancel</el-button>
        <el-button type="primary" :loading="generating" @click="submitGenerate">Generate</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ideaApi, paperApi, phase4Api } from "@/api";
import type { Idea, IdeaCreateRequest, IdeaDetail, Paper, Phase4DatasetProfile, Phase4Idea, Phase4IdeaScoreView } from "@/types/domain";
import { ElMessage, ElMessageBox } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { overallIdeaScore, sortPhase4Ideas, toPrettyJson } from "@/views/phase4/phase4Ui";

const activeTab = ref("stage3");

const ideas = ref<Idea[]>([]);
const papers = ref<Paper[]>([]);
const detail = ref<IdeaDetail>();
const loading = ref(false);
const detailLoading = ref(false);
const detailVisible = ref(false);
const createDialogVisible = ref(false);
const generateDialogVisible = ref(false);
const submitting = ref(false);
const generating = ref(false);
const selectedPaperId = ref("");
const editingId = ref("");

const phase4DatasetProfiles = ref<Phase4DatasetProfile[]>([]);
const phase4Ideas = ref<Phase4Idea[]>([]);
const phase4ScoreViews = ref<Phase4IdeaScoreView[]>([]);
const phase4Loading = ref(false);
const phase4Detail = ref<Phase4Idea>();
const phase4DetailVisible = ref(false);
const phase4ActionLoadingId = ref("");

const phase4Filters = reactive({
  datasetProfileId: "",
  status: "",
  sortBy: "overallScore",
  revisionOnly: false
});

const phase4Statuses = [
  "draft",
  "scored",
  "rejected",
  "selected",
  "implemented",
  "failed",
  "archived"
];

const phase4ScoreMap = computed(() => {
  const out: Record<string, number> = {};
  phase4ScoreViews.value.forEach((item) => {
    out[item.id] = item.overallScore;
  });
  return out;
});

const phase4RankMap = computed(() => {
  const out: Record<string, number> = {};
  phase4ScoreViews.value.forEach((item) => {
    out[item.id] = item.rank;
  });
  return out;
});

const filteredPhase4Ideas = computed(() => {
  const filtered = phase4Ideas.value.filter((item) => {
    if (phase4Filters.datasetProfileId && item.datasetProfileId !== phase4Filters.datasetProfileId) {
      return false;
    }
    if (phase4Filters.status && item.status !== phase4Filters.status) {
      return false;
    }
    if (phase4Filters.revisionOnly && !item.revisionOfId) {
      return false;
    }
    return true;
  });
  return sortPhase4Ideas(filtered, phase4Filters.sortBy, phase4ScoreMap.value);
});

const form = reactive<IdeaCreateRequest>({
  title: "",
  descriptionMd: "",
  status: "draft",
  weight: 70,
  priority: 70,
  confidence: 0.7,
  sourceType: "human",
  sourceNote: ""
});

onMounted(() => {
  void Promise.all([load(), loadPapers(), loadPhase4Ideas()]);
});

watch(activeTab, async (value) => {
  if (value === "phase4" && !phase4DatasetProfiles.value.length) {
    await loadPhase4Ideas();
  }
});

async function refreshActiveTab() {
  if (activeTab.value === "phase4") {
    await loadPhase4Ideas();
    return;
  }
  await load();
}

async function load() {
  loading.value = true;
  try {
    ideas.value = await ideaApi.getIdeas();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    loading.value = false;
  }
}

async function loadPapers() {
  papers.value = await paperApi.getPapers();
}

async function loadPhase4Ideas() {
  phase4Loading.value = true;
  try {
    const [profiles, items, scoreViews] = await Promise.all([
      phase4Api.getPhase4DatasetProfiles({ status: "active" }),
      phase4Api.getPhase4Ideas({
        datasetProfileId: phase4Filters.datasetProfileId || undefined,
        status: phase4Filters.status || undefined
      }),
      phase4Api.getPhase4IdeaScoreViews({
        datasetProfileId: phase4Filters.datasetProfileId || undefined,
        status: phase4Filters.status || undefined
      })
    ]);
    phase4DatasetProfiles.value = profiles;
    phase4Ideas.value = items;
    phase4ScoreViews.value = scoreViews;
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
    detail.value = await ideaApi.getIdeaById(id);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    detailLoading.value = false;
  }
}

function openPhase4Idea(item: Phase4Idea) {
  phase4Detail.value = item;
  phase4DetailVisible.value = true;
}

async function submitCreate() {
  submitting.value = true;
  try {
    if (editingId.value) {
      await ideaApi.updateIdea(editingId.value, { ...form });
      ElMessage.success("Stage3 idea updated.");
    } else {
      await ideaApi.createIdea(form);
      ElMessage.success("Stage3 idea created.");
    }
    createDialogVisible.value = false;
    resetForm();
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
  }
}

async function submitGenerate() {
  if (!selectedPaperId.value) {
    ElMessage.warning("Select a paper first.");
    return;
  }
  generating.value = true;
  try {
    const result = await ideaApi.generateIdeasFromPaper(selectedPaperId.value);
    ElMessage.success(`Generated ${result.ideas.length} stage3 ideas.`);
    generateDialogVisible.value = false;
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    generating.value = false;
  }
}

async function openEdit(id: string) {
  const result = await ideaApi.getIdeaById(id);
  editingId.value = id;
  form.title = result.idea.title;
  form.descriptionMd = result.idea.descriptionMd;
  form.status = result.idea.status;
  form.weight = result.idea.weight;
  form.priority = result.idea.priority;
  form.confidence = result.idea.confidence;
  form.sourceType = result.idea.sourceType;
  form.sourceNote = result.sources?.[0]?.sourceNote || "";
  createDialogVisible.value = true;
}

async function selectPhase4Idea(id: string) {
  await runPhase4IdeaAction(id, async () => {
    await phase4Api.selectPhase4Idea(id);
    ElMessage.success("Phase4 idea marked as selected.");
    await loadPhase4Ideas();
  });
}

async function rejectPhase4Idea(id: string) {
  await runPhase4IdeaAction(id, async () => {
    await phase4Api.rejectPhase4Idea(id);
    ElMessage.success("Phase4 idea rejected.");
    await loadPhase4Ideas();
  });
}

async function archivePhase4Idea(id: string) {
  await runPhase4IdeaAction(id, async () => {
    await phase4Api.archivePhase4Idea(id);
    ElMessage.success("Phase4 idea archived.");
    await loadPhase4Ideas();
  });
}

async function deletePhase4Idea(id: string) {
  try {
    await ElMessageBox.confirm("Delete this phase4 idea record?", "Confirm", { type: "warning" });
  } catch {
    return;
  }
  await runPhase4IdeaAction(id, async () => {
    await phase4Api.deletePhase4Idea(id);
    ElMessage.success("Phase4 idea deleted.");
    if (phase4Detail.value?.id === id) {
      phase4DetailVisible.value = false;
      phase4Detail.value = undefined;
    }
    await loadPhase4Ideas();
  });
}

async function runPhase4IdeaAction(id: string, action: () => Promise<void>) {
  phase4ActionLoadingId.value = id;
  try {
    await action();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    phase4ActionLoadingId.value = "";
  }
}

function datasetProfileName(datasetProfileId?: string) {
  return phase4DatasetProfiles.value.find((item) => item.id === datasetProfileId)?.datasetName || datasetProfileId || "-";
}

function displayOverallScore(item: Phase4Idea) {
  const score = phase4ScoreMap.value[item.id] || overallIdeaScore(item);
  return score ? score.toFixed(2) : "-";
}

function scoreRank(id: string) {
  return phase4RankMap.value[id];
}

function resetForm() {
  editingId.value = "";
  form.title = "";
  form.descriptionMd = "";
  form.status = "draft";
  form.weight = 70;
  form.priority = 70;
  form.confidence = 0.7;
  form.sourceType = "human";
  form.sourceNote = "";
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

.filter-row {
  margin-bottom: 12px;
}
</style>

<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">实验自动化</h1>
        <p class="page-subtitle">查看阶段2实验列表、当前状态和最近一次 run，作为实验调度与执行的最小入口。</p>
      </div>
      <el-space wrap>
        <el-button :loading="loading" @click="loadAll">刷新</el-button>
        <el-button type="primary" @click="dialogVisible = true">创建实验</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-card>
      <template #header>实验列表</template>
      <el-skeleton v-if="loading && !rows.length" :rows="6" animated />
      <el-empty v-else-if="!rows.length" description="暂无实验，可以先从 dataset asset 创建一个实验。" />
      <el-table v-else :data="rows" size="small">
        <el-table-column prop="title" label="Title" min-width="220" />
        <el-table-column label="Dataset" min-width="180">
          <template #default="{ row }">{{ row.datasetName }}</template>
        </el-table-column>
        <el-table-column label="Idea" min-width="180">
          <template #default="{ row }">{{ row.ideaTitle || "-" }}</template>
        </el-table-column>
        <el-table-column label="Status" width="120">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column prop="priority" label="Priority" width="90" />
        <el-table-column label="最近 Run" min-width="180">
          <template #default="{ row }">
            <div>{{ row.latestRun?.id || "-" }}</div>
            <div class="muted-text">
              <StatusTag v-if="row.latestRun" :status="row.latestRun.runStatus" />
              <span v-else>暂无 run</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="openDetail(row.id)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="创建实验" width="720px">
      <el-form :model="form" label-width="120px">
        <el-form-item label="Dataset Asset" required>
          <el-select v-model="form.datasetAssetId" filterable style="width: 100%">
            <el-option v-for="item in datasetAssets" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Idea">
          <el-select v-model="form.ideaId" filterable clearable style="width: 100%">
            <el-option v-for="item in ideas" :key="item.id" :label="item.title" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Baseline">
          <el-select v-model="form.baselineId" filterable clearable style="width: 100%">
            <el-option v-for="item in baselineOptions" :key="item.id" :label="item.name" :value="item.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="Title">
          <el-input v-model="form.title" placeholder="留空则自动生成" />
        </el-form-item>
        <el-form-item label="Priority">
          <el-input-number v-model="form.priority" :min="0" :max="100" style="width: 100%" />
        </el-form-item>
        <el-form-item label="Summary">
          <el-input v-model="form.summaryMd" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { experimentApi, ideaApi, researchAssetApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { Baseline, DatasetAsset, Experiment, ExperimentCreateRequest, ExperimentRun, Idea } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";

type ExperimentRow = Experiment & {
  datasetName: string;
  ideaTitle?: string;
  latestRun?: ExperimentRun;
};

const router = useRouter();

const loading = ref(false);
const submitting = ref(false);
const error = ref("");
const dialogVisible = ref(false);
const experiments = ref<Experiment[]>([]);
const datasetAssets = ref<DatasetAsset[]>([]);
const ideas = ref<Idea[]>([]);
const baselines = ref<Baseline[]>([]);
const runMap = ref<Record<string, ExperimentRun>>({});

const form = reactive<ExperimentCreateRequest>({
  datasetAssetId: "",
  ideaId: "",
  baselineId: "",
  title: "",
  priority: 50,
  summaryMd: "",
  ownerNoteMd: ""
});

const datasetMap = computed(() => Object.fromEntries(datasetAssets.value.map((item) => [item.id, item.name])));
const ideaMap = computed(() => Object.fromEntries(ideas.value.map((item) => [item.id, item.title])));
const baselineOptions = computed(() =>
  baselines.value.filter((item) => !form.datasetAssetId || item.datasetAssetId === form.datasetAssetId)
);
const rows = computed<ExperimentRow[]>(() =>
  experiments.value.map((item) => ({
    ...item,
    datasetName: datasetMap.value[item.datasetAssetId] || item.datasetAssetId,
    ideaTitle: item.ideaId ? ideaMap.value[item.ideaId] : "",
    latestRun: item.currentRunId ? runMap.value[item.currentRunId] : undefined
  }))
);

onMounted(() => {
  void loadAll();
});

async function loadAll() {
  loading.value = true;
  error.value = "";
  try {
    const [experimentList, assetList, ideaList, baselineList] = await Promise.all([
      experimentApi.getExperiments(),
      researchAssetApi.getDatasetAssets(),
      ideaApi.getIdeas(),
      researchAssetApi.getBaselines()
    ]);
    experiments.value = experimentList;
    datasetAssets.value = assetList;
    ideas.value = ideaList;
    baselines.value = baselineList;
    if (!form.datasetAssetId) {
      form.datasetAssetId = assetList[0]?.id || "";
    }
    const runIds = experimentList.map((item) => item.currentRunId).filter(Boolean) as string[];
    const runEntries = await Promise.all(
      runIds.map(async (id) => {
        try {
          return [id, await experimentApi.getRunById(id)] as const;
        } catch {
          return [id, undefined] as const;
        }
      })
    );
    runMap.value = Object.fromEntries(runEntries.filter((item): item is readonly [string, ExperimentRun] => Boolean(item[1])));
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

async function submitCreate() {
  if (!form.datasetAssetId) {
    ElMessage.warning("请先选择 dataset asset");
    return;
  }
  submitting.value = true;
  try {
    const detail = await experimentApi.createExperiment({
      ...form,
      ideaId: form.ideaId || undefined,
      baselineId: form.baselineId || undefined,
      title: form.title || undefined
    });
    ElMessage.success("实验已创建");
    dialogVisible.value = false;
    resetForm();
    await loadAll();
    await router.push(`/experiments/${detail.experiment.id}`);
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    submitting.value = false;
  }
}

function openDetail(id: string) {
  void router.push(`/experiments/${id}`);
}

function resetForm() {
  form.datasetAssetId = datasetAssets.value[0]?.id || "";
  form.ideaId = "";
  form.baselineId = "";
  form.title = "";
  form.priority = 50;
  form.summaryMd = "";
  form.ownerNoteMd = "";
}
</script>

<style scoped>
.page-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.muted-text {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-top: 4px;
}
</style>

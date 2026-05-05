<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">结果对比</h1>
        <p class="page-subtitle">查看 baseline 对比、历史结果对比和 summary，作为阶段2的最小结果评估视图。</p>
      </div>
      <el-space wrap>
        <el-button @click="refreshAll" :loading="loading">刷新</el-button>
        <el-button @click="backToExperiment">返回实验</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-row :gutter="12">
      <el-col :span="8">
        <el-card>
          <template #header>实验摘要</template>
          <el-skeleton v-if="loading && !detail" :rows="6" animated />
          <el-empty v-else-if="!detail" description="实验信息不可用" />
          <el-descriptions v-else :column="1" border>
            <el-descriptions-item label="Title">{{ detail.experiment.title }}</el-descriptions-item>
            <el-descriptions-item label="Dataset">{{ detail.datasetAsset.name }}</el-descriptions-item>
            <el-descriptions-item label="Baseline">{{ detail.baseline?.name || "-" }}</el-descriptions-item>
            <el-descriptions-item label="Current Run">{{ detail.experiment.currentRunId || "-" }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="16">
        <el-card>
          <template #header>Comparison 列表</template>
          <el-skeleton v-if="loading && !comparisons.length" :rows="6" animated />
          <el-empty v-else-if="!comparisons.length" description="暂无 comparison，可先在实验详情页触发 compare。" />
          <el-table v-else :data="comparisons" size="small" @row-click="selectComparison">
            <el-table-column label="Target" min-width="180">
              <template #default="{ row }">{{ targetLabel(row) }}</template>
            </el-table-column>
            <el-table-column label="Judgment" width="120">
              <template #default="{ row }">{{ judgment(row) }}</template>
            </el-table-column>
            <el-table-column label="Summary" min-width="220">
              <template #default="{ row }">{{ row.summaryMd }}</template>
            </el-table-column>
            <el-table-column label="Created" width="170">
              <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12" class="section-space">
      <el-col :span="24">
        <el-card>
          <template #header>对比详情</template>
          <el-empty v-if="!selected" description="点击一条 comparison 查看 baseline / 历史结果对比详情。" />
          <template v-else>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Target">{{ targetLabel(selected) }}</el-descriptions-item>
              <el-descriptions-item label="Judgment">{{ judgment(selected) }}</el-descriptions-item>
              <el-descriptions-item label="Summary">{{ selected.summaryMd }}</el-descriptions-item>
            </el-descriptions>
            <div class="section-space">
              <div class="subsection-title">指标逐项对比</div>
              <el-table :data="metricDiffs(selected)" size="small" empty-text="暂无指标差异">
                <el-table-column prop="metric" label="Metric" min-width="140" />
                <el-table-column prop="candidate_value" label="Current Run" width="120" />
                <el-table-column prop="target_value" label="Target" width="120" />
                <el-table-column prop="diff" label="Diff" width="120" />
                <el-table-column prop="judgment" label="判断" width="120" />
              </el-table>
            </div>
            <div class="section-space">
              <div class="subsection-title">Raw JSON</div>
              <pre class="pre-block">{{ pretty(selected.comparisonJson) }}</pre>
            </div>
          </template>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { experimentApi } from "@/api";
import type { ExperimentDetail, ResultComparison } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { computed, onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const error = ref("");
const detail = ref<ExperimentDetail>();
const comparisons = ref<ResultComparison[]>([]);
const selectedId = ref("");

const experimentId = computed(() => String(route.params.id || ""));
const selected = computed(() => comparisons.value.find((item) => item.id === selectedId.value) || comparisons.value[0]);

onMounted(() => {
  void refreshAll();
});

async function refreshAll() {
  loading.value = true;
  error.value = "";
  try {
    const [detailResult, comparisonList] = await Promise.all([
      experimentApi.getExperimentById(experimentId.value),
      experimentApi.getExperimentComparisons(experimentId.value)
    ]);
    detail.value = detailResult;
    comparisons.value = comparisonList;
    selectedId.value = comparisonList[0]?.id || "";
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

function selectComparison(item: ResultComparison) {
  selectedId.value = item.id;
}

function targetLabel(item: ResultComparison) {
  if (item.baselineId) {
    return `Baseline · ${item.baselineId}`;
  }
  if (item.targetResultArchiveId) {
    return `历史结果 · ${item.targetResultArchiveId}`;
  }
  return item.id;
}

function judgment(item: ResultComparison) {
  return (item.comparisonJson?.judgment as string) || "-";
}

function metricDiffs(item: ResultComparison) {
  return Array.isArray(item.comparisonJson?.metric_diffs) ? (item.comparisonJson.metric_diffs as Record<string, unknown>[]) : [];
}

function pretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

function backToExperiment() {
  void router.push(`/experiments/${experimentId.value}`);
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
  font-weight: 600;
  margin-bottom: 10px;
}

.pre-block {
  white-space: pre-wrap;
  word-break: break-word;
  background: var(--panel-alt);
  border: 1px solid var(--border);
  padding: 12px;
  border-radius: 8px;
  max-height: 420px;
  overflow: auto;
}
</style>

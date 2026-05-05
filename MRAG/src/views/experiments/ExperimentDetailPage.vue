<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">{{ detail?.experiment.title || "实验详情" }}</h1>
        <p class="page-subtitle">查看 spec、执行状态、日志和恢复信息，并触发 generate spec / queue / schedule / start / retry。</p>
      </div>
      <el-space wrap>
        <el-button @click="refreshAll" :loading="loading">刷新</el-button>
        <el-button @click="openComparisons" :disabled="!experimentId">结果对比</el-button>
        <el-button type="primary" @click="generateSpec" :loading="actionLoading.generateSpec">生成 Spec</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-skeleton v-if="loading && !detail" :rows="10" animated />
    <el-empty v-else-if="!detail" description="实验不存在或尚未加载。" />
    <template v-else>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-card>
            <template #header>基础信息</template>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Dataset">{{ detail.datasetAsset.name }}</el-descriptions-item>
              <el-descriptions-item label="Idea">{{ detail.idea?.title || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Baseline">{{ detail.baseline?.name || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Status"><StatusTag :status="detail.experiment.status" /></el-descriptions-item>
              <el-descriptions-item label="Priority">{{ detail.experiment.priority }}</el-descriptions-item>
              <el-descriptions-item label="更新时间">{{ formatDateTime(detail.experiment.updatedAt) }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>实验动作</template>
            <el-space wrap>
              <el-button @click="generateSpec" :loading="actionLoading.generateSpec">Generate Spec</el-button>
              <el-button @click="queue" :loading="actionLoading.queue">Queue</el-button>
              <el-button @click="schedule" :loading="actionLoading.schedule" :disabled="!currentRun?.id">Schedule</el-button>
              <el-button type="primary" @click="startRun" :loading="actionLoading.start" :disabled="!currentRun?.id">Start Run</el-button>
              <el-button type="warning" @click="retryRun" :loading="actionLoading.retry" :disabled="currentRun?.runStatus !== 'failed'">Retry</el-button>
              <el-button @click="runCompare" :loading="actionLoading.compare" :disabled="currentRun?.runStatus !== 'succeeded'">Compare</el-button>
            </el-space>
            <div class="section-space">
              <el-empty v-if="!currentRun" description="当前还没有 run，先 queue 一个 run。" />
              <el-descriptions v-else :column="1" border>
                <el-descriptions-item label="Run ID">{{ currentRun.id }}</el-descriptions-item>
                <el-descriptions-item label="Run Status"><StatusTag :status="currentRun.runStatus" /></el-descriptions-item>
                <el-descriptions-item label="Assigned Server">{{ currentRun.assignedServerId || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Started At">{{ formatDateTime(currentRun.startedAt) }}</el-descriptions-item>
                <el-descriptions-item label="Ended At">{{ formatDateTime(currentRun.endedAt) }}</el-descriptions-item>
                <el-descriptions-item label="Error">{{ currentRun.errorMessage || "-" }}</el-descriptions-item>
              </el-descriptions>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>Spec</template>
            <el-empty v-if="!spec" description="Spec 尚未生成。" />
            <template v-else>
              <div class="muted-text">模板：{{ spec.spec.templateType }} · 版本：v{{ spec.spec.version }}</div>
              <pre class="pre-block">{{ pretty(spec.spec.specJson) }}</pre>
            </template>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>恢复信息</template>
            <el-empty v-if="!recovery" description="当前没有失败恢复信息。" />
            <el-descriptions v-else :column="1" border>
              <el-descriptions-item label="失败阶段">{{ recovery.failureStage }}</el-descriptions-item>
              <el-descriptions-item label="失败原因">{{ recovery.failureReason }}</el-descriptions-item>
              <el-descriptions-item label="建议重试">{{ recovery.suggestRetry ? "是" : "否" }}</el-descriptions-item>
              <el-descriptions-item label="日志摘要">{{ recovery.lastLogSummary || "-" }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>运行详情 / 日志</template>
            <el-empty v-if="!currentRun" description="暂无运行信息。" />
            <template v-else>
              <el-radio-group v-model="logType" size="small" class="section-space-sm">
                <el-radio-button label="stdout">stdout</el-radio-button>
                <el-radio-button label="stderr">stderr</el-radio-button>
              </el-radio-group>
              <div class="log-actions">
                <el-button size="small" @click="loadLogs">刷新日志</el-button>
              </div>
              <pre class="pre-block log-block">{{ logTail || "暂无 tail 日志" }}</pre>
              <el-table :data="logs" size="small" empty-text="暂无日志记录">
                <el-table-column prop="logType" label="Type" width="100" />
                <el-table-column prop="logPath" label="Path" min-width="220" />
                <el-table-column label="更新时间" width="170">
                  <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
                </el-table-column>
              </el-table>
            </template>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>调度与对比摘要</template>
            <el-empty v-if="!currentRun" description="暂无调度信息。" />
            <template v-else>
              <el-descriptions :column="1" border>
                <el-descriptions-item label="Chosen Server">{{ decision?.chosenServerId || "-" }}</el-descriptions-item>
                <el-descriptions-item label="Decision JSON">
                  <pre class="pre-inline">{{ pretty(decision?.decisionJson || {}) }}</pre>
                </el-descriptions-item>
              </el-descriptions>
              <div class="section-space">
                <div class="subsection-title">最近 Comparison</div>
                <el-empty v-if="!comparisons.length" description="还没有 comparison。" />
                <el-table v-else :data="comparisons.slice(0, 5)" size="small">
                  <el-table-column label="Target" min-width="140">
                    <template #default="{ row }">{{ comparisonTarget(row) }}</template>
                  </el-table-column>
                  <el-table-column label="判断" width="120">
                    <template #default="{ row }">{{ comparisonJudgment(row) }}</template>
                  </el-table-column>
                  <el-table-column label="摘要" min-width="180">
                    <template #default="{ row }">{{ row.summaryMd }}</template>
                  </el-table-column>
                </el-table>
              </div>
            </template>
          </el-card>
        </el-col>
      </el-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { experimentApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type {
  ExperimentDetail,
  ExperimentRun,
  ExperimentSpecDetail,
  ResultComparison,
  RunLog,
  RunRecoveryDetail,
  SchedulerDecision
} from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const loading = ref(false);
const error = ref("");
const detail = ref<ExperimentDetail>();
const spec = ref<ExperimentSpecDetail>();
const currentRun = ref<ExperimentRun>();
const recovery = ref<RunRecoveryDetail>();
const logs = ref<RunLog[]>([]);
const logTail = ref("");
const logType = ref("stdout");
const decision = ref<SchedulerDecision>();
const comparisons = ref<ResultComparison[]>([]);

const actionLoading = reactive({
  generateSpec: false,
  queue: false,
  schedule: false,
  start: false,
  retry: false,
  compare: false
});

const experimentId = computed(() => String(route.params.id || ""));

onMounted(() => {
  void refreshAll();
});

watch(logType, () => {
  if (currentRun.value?.id) {
    void loadLogs();
  }
});

async function refreshAll() {
  if (!experimentId.value) {
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    detail.value = await experimentApi.getExperimentById(experimentId.value);
    spec.value = await safeLoad(() => experimentApi.getExperimentSpec(experimentId.value));
    comparisons.value = await safeLoad(() => experimentApi.getExperimentComparisons(experimentId.value), []);
    if (detail.value.experiment.currentRunId) {
      currentRun.value = await safeLoad(() => experimentApi.getRunById(detail.value!.experiment.currentRunId!), undefined);
      if (currentRun.value?.id) {
        decision.value = await safeLoad(() => experimentApi.getSchedulerDecision(currentRun.value!.id), undefined);
        recovery.value = currentRun.value.runStatus === "failed"
          ? await safeLoad(() => experimentApi.getRunRecovery(currentRun.value!.id), undefined)
          : undefined;
        await loadLogs();
      } else {
        logs.value = [];
        logTail.value = "";
      }
    } else {
      currentRun.value = undefined;
      decision.value = undefined;
      recovery.value = undefined;
      logs.value = [];
      logTail.value = "";
    }
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

async function generateSpec() {
  await withAction("generateSpec", async () => {
    spec.value = await experimentApi.generateExperimentSpec(experimentId.value);
    ElMessage.success("Spec 已生成");
    await refreshAll();
  });
}

async function queue() {
  await withAction("queue", async () => {
    const result = await experimentApi.queueExperiment(experimentId.value);
    currentRun.value = result.run;
    ElMessage.success("已进入 queued");
    await refreshAll();
  });
}

async function schedule() {
  if (!currentRun.value?.id) {
    return;
  }
  await withAction("schedule", async () => {
    const result = await experimentApi.scheduleRun(currentRun.value!.id);
    currentRun.value = result.run;
    decision.value = result.decision;
    ElMessage.success(`已调度到 ${result.chosen.serverName}`);
    await refreshAll();
  });
}

async function startRun() {
  if (!currentRun.value?.id) {
    return;
  }
  await withAction("start", async () => {
    currentRun.value = await experimentApi.startRun(currentRun.value!.id);
    ElMessage.success("Run 启动完成");
    await refreshAll();
  });
}

async function retryRun() {
  if (!currentRun.value?.id) {
    return;
  }
  await withAction("retry", async () => {
    const result = await experimentApi.retryRun(currentRun.value!.id);
    currentRun.value = result.run;
    ElMessage.success("已创建重试 run 并重新进入 queued");
    await refreshAll();
  });
}

async function runCompare() {
  if (!currentRun.value?.id) {
    return;
  }
  await withAction("compare", async () => {
    await experimentApi.compareRun(currentRun.value!.id);
    ElMessage.success("结果对比已生成");
    await refreshAll();
  });
}

async function loadLogs() {
  if (!currentRun.value?.id) {
    return;
  }
  logs.value = await safeLoad(() => experimentApi.getRunLogs(currentRun.value!.id), []);
  const tail = await safeLoad(() => experimentApi.getRunLogTail(currentRun.value!.id, logType.value), { tail: "" });
  logTail.value = tail.tail;
}

function openComparisons() {
  void router.push(`/experiments/${experimentId.value}/comparisons`);
}

function pretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

function comparisonTarget(item: ResultComparison) {
  if (item.baselineId) {
    return `Baseline: ${item.baselineId}`;
  }
  if (item.targetResultArchiveId) {
    return `Archive: ${item.targetResultArchiveId}`;
  }
  return item.id;
}

function comparisonJudgment(item: ResultComparison) {
  return (item.comparisonJson?.judgment as string) || "-";
}

async function withAction(
  key: keyof typeof actionLoading,
  action: () => Promise<void>
) {
  actionLoading[key] = true;
  try {
    await action();
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    actionLoading[key] = false;
  }
}

async function safeLoad<T>(loader: () => Promise<T>, fallback?: T): Promise<T> {
  try {
    return await loader();
  } catch {
    return fallback as T;
  }
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
  margin-bottom: 8px;
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

.pre-inline {
  margin: 0;
  white-space: pre-wrap;
}

.log-block {
  min-height: 180px;
}

.log-actions {
  margin-bottom: 8px;
}

.section-space-sm {
  margin-bottom: 8px;
}

.subsection-title {
  font-weight: 600;
  margin-bottom: 10px;
}
</style>

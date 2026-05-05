<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Agent List</h1>
        <p class="page-subtitle">Inspect controlled agents, latest jobs, input/output summaries, and a minimal retrigger entry.</p>
      </div>
      <el-space wrap>
        <el-button :loading="loading" @click="loadAgents">Refresh</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-card>
      <template #header>Controlled Agents</template>
      <el-table :data="items" v-loading="loading" size="small" empty-text="No agent data">
        <el-table-column prop="agent_type" label="Agent" width="140" />
        <el-table-column label="Subscriptions" min-width="180">
          <template #default="{ row }">
            <el-space wrap>
              <el-tag v-for="item in row.event_types" :key="item" size="small">{{ item }}</el-tag>
              <span v-if="!row.event_types.length" class="muted-text">-</span>
            </el-space>
          </template>
        </el-table-column>
        <el-table-column label="Execution" min-width="180">
          <template #default="{ row }">
            <div>{{ row.execution_mode || "-" }}</div>
            <div class="muted-text">{{ formatModel(row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="Latest Status" width="170">
          <template #default="{ row }">
            <StatusTag v-if="row.latest_job" :status="row.latest_job.status" />
            <span v-else class="muted-text">No jobs</span>
            <div v-if="row.latest_job" class="meta-stack">
              <span>Validation: {{ row.latest_job.validation_status || "-" }}</span>
              <span>Repair: {{ row.latest_job.repair_status || "-" }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Latest Input" min-width="220">
          <template #default="{ row }">{{ formatAgentInputSummary(row.latest_job?.input_refs) }}</template>
        </el-table-column>
        <el-table-column label="Latest Output" min-width="260">
          <template #default="{ row }">{{ formatAgentOutputSummary(row.latest_job?.normalized_payload) }}</template>
        </el-table-column>
        <el-table-column label="Tools / Skills" min-width="220">
          <template #default="{ row }">
            <div class="meta-stack">
              <span>Tools: {{ formatTagList(row.tool_refs).join(", ") || "-" }}</span>
              <span>Skills: {{ formatTagList(row.skill_refs).join(", ") || "-" }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Control" width="130">
          <template #default="{ row }">
            <div>Concurrency {{ row.concurrency_limit || 0 }}</div>
            <div class="muted-text">Retries {{ row.max_retries || 0 }}</div>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="170" fixed="right">
          <template #default="{ row }">
            <el-space wrap>
              <el-button text type="primary" :disabled="!row.latest_job" @click="openJob(row.latest_job?.id)">
                Open Job
              </el-button>
              <el-button
                text
                type="warning"
                :disabled="!row.latest_job"
                :loading="triggeringId === row.latest_job?.id"
                @click="triggerLatest(row.latest_job?.id)"
              >
                Retrigger
              </el-button>
            </el-space>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { agentApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { AgentSummary } from "@/types/domain";
import { formatAgentInputSummary, formatAgentOutputSummary, formatTagList } from "@/views/agents/helpers";
import { ElMessage } from "element-plus";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const loading = ref(false);
const error = ref("");
const triggeringId = ref("");
const items = ref<AgentSummary[]>([]);

onMounted(() => {
  void loadAgents();
});

async function loadAgents() {
  loading.value = true;
  error.value = "";
  try {
    items.value = await agentApi.getAgents();
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

function formatModel(row: AgentSummary) {
  const provider = row.model_provider || "-";
  const modelName = row.model_name || "-";
  return `${provider} / ${modelName}`;
}

function openJob(id?: string) {
  if (!id) {
    return;
  }
  void router.push(`/agents/jobs/${id}`);
}

async function triggerLatest(id?: string) {
  if (!id) {
    return;
  }
  triggeringId.value = id;
  try {
    await agentApi.triggerAgentJob(id);
    ElMessage.success("Retriggered the latest agent job");
    await loadAgents();
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    triggeringId.value = "";
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
}

.meta-stack {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  margin-top: 4px;
}
</style>

<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Agent Job List</h1>
        <p class="page-subtitle">Review recent jobs, validation and repair status, input/output summaries, and retrigger control.</p>
      </div>
      <el-space wrap>
        <el-select v-model="selectedAgentType" clearable placeholder="Filter agent" style="width: 160px">
          <el-option v-for="item in agentTypes" :key="item" :label="item" :value="item" />
        </el-select>
        <el-select v-model="selectedStatus" clearable placeholder="Filter status" style="width: 140px">
          <el-option v-for="item in statusOptions" :key="item" :label="item" :value="item" />
        </el-select>
        <el-button :loading="loading" @click="loadJobs">Refresh</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-card>
      <template #header>Recent Jobs</template>
      <el-table :data="filteredJobs" v-loading="loading" size="small" empty-text="No agent jobs">
        <el-table-column prop="id" label="Job ID" min-width="200" />
        <el-table-column prop="agent_type" label="Agent" width="130" />
        <el-table-column label="Execution" min-width="180">
          <template #default="{ row }">
            <div>{{ row.execution_mode || "-" }}</div>
            <div class="muted-text">{{ row.model_provider || "-" }} / {{ row.model_name || "-" }}</div>
          </template>
        </el-table-column>
        <el-table-column label="Status" width="170">
          <template #default="{ row }">
            <StatusTag :status="row.status" />
            <div class="meta-stack">
              <span>Validation: {{ row.validation_status || "-" }}</span>
              <span>Repair: {{ row.repair_status || "-" }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="Input Summary" min-width="220">
          <template #default="{ row }">{{ formatAgentInputSummary(row.input_refs) }}</template>
        </el-table-column>
        <el-table-column label="Output Summary" min-width="240">
          <template #default="{ row }">{{ formatAgentOutputSummary(row.normalized_payload) }}</template>
        </el-table-column>
        <el-table-column label="Updated At" width="170">
          <template #default="{ row }">{{ formatDateTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="Actions" width="170" fixed="right">
          <template #default="{ row }">
            <el-space wrap>
              <el-button text type="primary" @click="openJob(row.id)">Details</el-button>
              <el-button
                text
                type="warning"
                :loading="triggeringId === row.id"
                @click="triggerJob(row.id)"
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
import type { AgentJob } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { formatAgentInputSummary, formatAgentOutputSummary } from "@/views/agents/helpers";
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const jobs = ref<AgentJob[]>([]);
const loading = ref(false);
const error = ref("");
const triggeringId = ref("");
const selectedAgentType = ref("");
const selectedStatus = ref("");

const agentTypes = computed(() => Array.from(new Set(jobs.value.map((item) => item.agent_type))).sort());
const statusOptions = computed(() => Array.from(new Set(jobs.value.map((item) => item.status))).sort());
const filteredJobs = computed(() =>
  jobs.value.filter((item) => {
    if (selectedAgentType.value && item.agent_type !== selectedAgentType.value) {
      return false;
    }
    if (selectedStatus.value && item.status !== selectedStatus.value) {
      return false;
    }
    return true;
  })
);

onMounted(() => {
  void loadJobs();
});

async function loadJobs() {
  loading.value = true;
  error.value = "";
  try {
    jobs.value = await agentApi.getAgentJobs();
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

function openJob(id: string) {
  void router.push(`/agents/jobs/${id}`);
}

async function triggerJob(id: string) {
  triggeringId.value = id;
  try {
    await agentApi.triggerAgentJob(id);
    ElMessage.success("Agent job retriggered");
    await loadJobs();
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

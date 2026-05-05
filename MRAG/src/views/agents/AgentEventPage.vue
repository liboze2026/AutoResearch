<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Pipeline Events</h1>
        <p class="page-subtitle">Inspect database-driven agent events, source refs, event state, and triggered jobs.</p>
      </div>
      <el-space wrap>
        <el-button :loading="loading" @click="loadEvents">Refresh</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-card>
      <template #header>Recent Events</template>
      <el-table :data="events" v-loading="loading" size="small" empty-text="No events">
        <el-table-column prop="event_type" label="Event" width="170" />
        <el-table-column prop="source_ref" label="Source Ref" min-width="220" />
        <el-table-column prop="status" label="Status" width="130">
          <template #default="{ row }"><StatusTag :status="row.status" /></template>
        </el-table-column>
        <el-table-column label="Triggered Jobs" min-width="220">
          <template #default="{ row }">
            <el-space wrap>
              <el-button
                v-for="jobId in row.triggered_job_ids"
                :key="jobId"
                text
                type="primary"
                @click="openJob(jobId)"
              >
                {{ jobId }}
              </el-button>
              <span v-if="!row.triggered_job_ids.length" class="muted-text">-</span>
            </el-space>
          </template>
        </el-table-column>
        <el-table-column label="Payload Summary" min-width="260">
          <template #default="{ row }">{{ truncateJson(row.payload) }}</template>
        </el-table-column>
        <el-table-column label="Created At" width="170">
          <template #default="{ row }">{{ formatDateTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { agentApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { AgentEvent } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { truncateJson } from "@/views/agents/helpers";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const loading = ref(false);
const error = ref("");
const events = ref<AgentEvent[]>([]);

onMounted(() => {
  void loadEvents();
});

async function loadEvents() {
  loading.value = true;
  error.value = "";
  try {
    events.value = await agentApi.getAgentEvents();
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

function openJob(id: string) {
  void router.push(`/agents/jobs/${id}`);
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
</style>

<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">{{ job?.agent_type || "Agent Job" }} Details</h1>
        <p class="page-subtitle">Inspect the normalized contract, validation/repair status, artifacts, tool usage, and repair logs.</p>
      </div>
      <el-space wrap>
        <el-button :loading="loading" @click="loadAll">Refresh</el-button>
        <el-button type="warning" :loading="triggering" :disabled="!job?.id" @click="triggerJob">Retrigger</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-skeleton v-if="loading && !job" :rows="10" animated />
    <el-empty v-else-if="!job" description="Agent job not found" />
    <template v-else>
      <el-row :gutter="12">
        <el-col :span="12">
          <el-card>
            <template #header>Overview</template>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Job ID">{{ job.id }}</el-descriptions-item>
              <el-descriptions-item label="Agent">{{ job.agent_type }}</el-descriptions-item>
              <el-descriptions-item label="Status"><StatusTag :status="job.status" /></el-descriptions-item>
              <el-descriptions-item label="Execution">{{ job.execution_mode || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Model">{{ job.model_provider || "-" }} / {{ job.model_name || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Prompt">{{ job.prompt_version || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Schema">{{ job.output_schema_ref || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Workspace">{{ job.workspace_dir || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Updated At">{{ formatDateTime(job.updated_at) }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>Validation / Repair</template>
            <el-descriptions :column="1" border>
              <el-descriptions-item label="Validation">{{ job.validation_status || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Repair">{{ job.repair_status || "-" }}</el-descriptions-item>
              <el-descriptions-item label="Warnings">
                <div v-if="job.warnings.length" class="stack-list">
                  <div v-for="item in job.warnings" :key="item">{{ item }}</div>
                </div>
                <span v-else>-</span>
              </el-descriptions-item>
              <el-descriptions-item label="Validation Errors">
                <div v-if="job.validation_errors.length" class="stack-list">
                  <div v-for="item in job.validation_errors" :key="item">{{ item }}</div>
                </div>
                <span v-else>-</span>
              </el-descriptions-item>
              <el-descriptions-item label="Error">{{ job.error_message || "-" }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>Input Refs</template>
            <el-table :data="job.input_refs" size="small" empty-text="No input refs">
              <el-table-column prop="ref_type" label="Type" width="120" />
              <el-table-column prop="ref_id" label="Ref ID" min-width="160" />
              <el-table-column prop="ref_path" label="Path" min-width="220" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>Artifacts</template>
            <el-table :data="artifacts" size="small" empty-text="No artifacts">
              <el-table-column prop="artifact_type" label="Type" width="120" />
              <el-table-column prop="name" label="Name" min-width="150" />
              <el-table-column prop="file_path" label="Path" min-width="220" />
            </el-table>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>Normalized Output</template>
            <pre class="pre-block">{{ pretty(job.normalized_payload) }}</pre>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>Metadata</template>
            <pre class="pre-block">{{ pretty(job.metadata) }}</pre>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="12" class="section-space">
        <el-col :span="12">
          <el-card>
            <template #header>Repair Log</template>
            <el-table :data="job.repair_actions" size="small" empty-text="No repair actions">
              <el-table-column prop="action" label="Action" width="140" />
              <el-table-column prop="status" label="Status" width="110" />
              <el-table-column prop="detail" label="Detail" min-width="220" />
            </el-table>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card>
            <template #header>Tool Usages / Refs</template>
            <div class="section-block">
              <div class="subsection-title">Tool Usages</div>
              <el-table :data="job.tool_usages" size="small" empty-text="No tool usages">
                <el-table-column prop="tool_ref" label="Tool" min-width="180" />
                <el-table-column prop="status" label="Status" width="110" />
                <el-table-column prop="summary" label="Summary" min-width="180" />
              </el-table>
            </div>
            <div class="section-block">
              <div class="subsection-title">Refs</div>
              <div class="stack-list">
                <div>Tools: {{ job.tool_refs.join(", ") || "-" }}</div>
                <div>Skills: {{ job.skill_refs.join(", ") || "-" }}</div>
                <div>Memory: {{ job.memory_refs.join(", ") || "-" }}</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { agentApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { AgentArtifact, AgentJob } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
const loading = ref(false);
const triggering = ref(false);
const error = ref("");
const job = ref<AgentJob>();
const artifacts = ref<AgentArtifact[]>([]);

onMounted(() => {
  void loadAll();
});

watch(
  () => route.params.id,
  () => {
    void loadAll();
  }
);

async function loadAll() {
  const jobId = String(route.params.id || "");
  if (!jobId) {
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    const [jobData, artifactData] = await Promise.all([
      agentApi.getAgentJobById(jobId),
      agentApi.getAgentArtifacts(jobId)
    ]);
    job.value = jobData;
    artifacts.value = artifactData;
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
  }
}

async function triggerJob() {
  if (!job.value?.id) {
    return;
  }
  triggering.value = true;
  try {
    await agentApi.triggerAgentJob(job.value.id);
    ElMessage.success("Agent job retriggered");
    await loadAll();
  } catch (err) {
    ElMessage.error((err as Error).message);
  } finally {
    triggering.value = false;
  }
}

function pretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}
</script>

<style scoped>
.page-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
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
  margin: 0;
}

.stack-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.section-block + .section-block {
  margin-top: 16px;
}

.subsection-title {
  font-weight: 600;
  margin-bottom: 10px;
}
</style>

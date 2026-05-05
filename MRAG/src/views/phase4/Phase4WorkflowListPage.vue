<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Phase4 Workflow</h1>
        <p class="page-subtitle">Create dataset-driven phase4 workflows and stop at the manual idea selection gate before coding and writing continue.</p>
      </div>
      <el-space wrap>
        <el-button @click="refreshAll" :loading="loading">Refresh</el-button>
      </el-space>
    </div>

    <el-row :gutter="12">
      <el-col :span="8">
        <el-card>
          <template #header>Create Workflow</template>
          <el-form label-width="110px">
            <el-form-item label="Dataset Profile" required>
              <el-select v-model="createForm.datasetProfileId" placeholder="Select phase4 dataset profile" filterable style="width: 100%">
                <el-option
                  v-for="item in datasetProfiles"
                  :key="item.id"
                  :label="`${item.datasetName} (${item.serverName || item.serverId || 'server'})`"
                  :value="item.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="Reader Mode">
              <el-select v-model="createForm.readerExecutionMode" style="width: 100%">
                <el-option label="api" value="api" />
                <el-option label="mock" value="mock" />
                <el-option label="codex_cli" value="codex_cli" />
              </el-select>
            </el-form-item>
            <el-form-item label="Idea Mode">
              <el-select v-model="createForm.ideaExecutionMode" style="width: 100%">
                <el-option label="api" value="api" />
                <el-option label="mock" value="mock" />
                <el-option label="codex_cli" value="codex_cli" />
              </el-select>
            </el-form-item>
            <el-form-item label="Coding Mode">
              <el-select v-model="createForm.codingExecutionMode" style="width: 100%">
                <el-option label="api" value="api" />
                <el-option label="mock" value="mock" />
                <el-option label="codex_cli" value="codex_cli" />
              </el-select>
            </el-form-item>
            <el-form-item label="Writer Mode">
              <el-select v-model="createForm.writerExecutionMode" style="width: 100%">
                <el-option label="api" value="api" />
                <el-option label="mock" value="mock" />
                <el-option label="codex_cli" value="codex_cli" />
              </el-select>
            </el-form-item>
            <el-form-item label="Runner">
              <el-select v-model="createForm.runnerMode" style="width: 100%">
                <el-option label="shenzhenvlab" value="shenzhenvlab" />
                <el-option label="local_dummy" value="local_dummy" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="creating" @click="createWorkflow">Create and Advance to Idea Pool</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="16">
        <el-card>
          <template #header>Workflow List</template>
          <el-row :gutter="12" class="filter-row">
            <el-col :span="10">
              <el-select v-model="filters.datasetProfileId" placeholder="Filter by dataset" clearable filterable style="width: 100%">
                <el-option v-for="item in datasetProfiles" :key="item.id" :label="item.datasetName" :value="item.id" />
              </el-select>
            </el-col>
            <el-col :span="8">
              <el-select v-model="filters.status" placeholder="Filter by status" clearable style="width: 100%">
                <el-option v-for="item in workflowStatuses" :key="item" :label="item" :value="item" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-button type="primary" style="width: 100%" :loading="loading" @click="loadWorkflows">Apply Filters</el-button>
            </el-col>
          </el-row>
          <el-table :data="workflows" size="small" v-loading="loading" empty-text="No phase4 workflow">
            <el-table-column label="Dataset" min-width="180">
              <template #default="{ row }">{{ datasetName(row.datasetProfileId) }}</template>
            </el-table-column>
            <el-table-column prop="status" label="Status" width="190" />
            <el-table-column prop="nextAction" label="Next Action" width="160" />
            <el-table-column label="Selected Idea" min-width="180">
              <template #default="{ row }">{{ String(row.metadata?.selected_idea_id || row.selectedIdeaId || "-") }}</template>
            </el-table-column>
            <el-table-column label="Updated" width="170">
              <template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template>
            </el-table-column>
            <el-table-column label="Actions" width="120">
              <template #default="{ row }">
                <el-button text type="primary" @click="openWorkflow(row.id)">Open</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { phase4Api } from "@/api";
import type { Phase4DatasetProfile, Phase4Workflow } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const datasetProfiles = ref<Phase4DatasetProfile[]>([]);
const workflows = ref<Phase4Workflow[]>([]);
const loading = ref(false);
const creating = ref(false);

const filters = reactive({
  datasetProfileId: String(route.query.datasetProfileId || ""),
  status: String(route.query.status || "")
});

const createForm = reactive({
  datasetProfileId: String(route.query.datasetProfileId || ""),
  readerExecutionMode: "api",
  ideaExecutionMode: "api",
  codingExecutionMode: "api",
  writerExecutionMode: "api",
  runnerMode: "shenzhenvlab"
});

const workflowStatuses = computed(() => [
  "running_reader",
  "running_idea",
  "awaiting_selection",
  "running_coding",
  "awaiting_revision_selection",
  "running_writing",
  "completed",
  "blocked",
  "archived"
]);

onMounted(async () => {
  await refreshAll();
});

async function refreshAll() {
  loading.value = true;
  try {
    datasetProfiles.value = await phase4Api.getPhase4DatasetProfiles({ status: "active" });
    if (!createForm.datasetProfileId) {
      createForm.datasetProfileId = filters.datasetProfileId || datasetProfiles.value[0]?.id || "";
    }
    await loadWorkflows();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    loading.value = false;
  }
}

async function loadWorkflows() {
  workflows.value = await phase4Api.getPhase4Workflows({
    datasetProfileId: filters.datasetProfileId || undefined,
    status: filters.status || undefined
  });
}

async function createWorkflow() {
  if (!createForm.datasetProfileId) {
    ElMessage.warning("Select a phase4 dataset profile first.");
    return;
  }
  const profile = datasetProfiles.value.find((item) => item.id === createForm.datasetProfileId);
  creating.value = true;
  try {
    const detail = await phase4Api.createPhase4Workflow({
      datasetProfileId: createForm.datasetProfileId,
      reader: {
        executionMode: createForm.readerExecutionMode
      },
      idea: {
        executionMode: createForm.ideaExecutionMode
      },
      coding: {
        executionMode: createForm.codingExecutionMode,
        runnerMode: createForm.runnerMode,
        serverId: createForm.runnerMode === "shenzhenvlab" ? profile?.serverId : undefined
      },
      writing: {
        executionMode: createForm.writerExecutionMode
      }
    });
    ElMessage.success("Phase4 workflow created.");
    await router.push(`/phase4/workflows/${detail.workflow.id}`);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    creating.value = false;
  }
}

function openWorkflow(id: string) {
  void router.push(`/phase4/workflows/${id}`);
}

function datasetName(datasetProfileId: string) {
  return datasetProfiles.value.find((item) => item.id === datasetProfileId)?.datasetName || datasetProfileId;
}
</script>

<style scoped>
.page-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.filter-row {
  margin-bottom: 12px;
}
</style>

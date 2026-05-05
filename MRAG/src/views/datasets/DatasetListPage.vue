<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">数据集管理</h1>
        <p class="page-subtitle">以服务器目录扫描为主流程，前端负责发现候选数据集、校验路径并完成登记。</p>
      </div>
      <el-space wrap>
        <el-button @click="load">刷新列表</el-button>
        <el-button type="primary" @click="openImportDialog()">手动登记</el-button>
      </el-space>
    </div>

    <el-row :gutter="12">
      <el-col :span="10">
        <el-card>
          <template #header>服务器扫描</template>
          <el-form :model="scanForm" label-width="96px">
            <el-form-item label="服务器" required>
              <el-select v-model="scanForm.serverId" placeholder="选择服务器" style="width: 100%">
                <el-option v-for="server in servers" :key="server.id" :label="`${server.name} (${server.host})`" :value="server.id" />
              </el-select>
            </el-form-item>
            <el-form-item label="扫描目录">
              <el-input v-model="scanForm.rootPath" placeholder="默认使用服务器配置中的 remoteRoot" />
            </el-form-item>
            <el-form-item label="最大深度">
              <el-input-number v-model="scanForm.maxDepth" :min="1" :max="6" style="width: 100%" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="scanLoading" @click="scanDatasets">扫描候选数据集</el-button>
            </el-form-item>
          </el-form>
          <el-alert
            v-if="scanResult"
            class="section-space"
            :title="`已从 ${scanResult.serverName || scanResult.serverId} 扫描到 ${scanResult.candidates.length} 个候选目录`"
            :description="`根目录：${scanResult.rootPath} | 模式：${scanResult.mode} | 时间：${formatDateTime(scanResult.scannedAt)}`"
            type="info"
            :closable="false"
            show-icon
          />
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card>
          <template #header>已登记数据集</template>
          <el-row :gutter="12" class="filter-row">
            <el-col :span="10"><el-input v-model="filters.keyword" placeholder="按名称或标签搜索" clearable /></el-col>
            <el-col :span="7">
              <el-select v-model="filters.sourceType" placeholder="来源" clearable style="width: 100%">
                <el-option label="本地" value="local" />
                <el-option label="服务器" value="remote" />
              </el-select>
            </el-col>
            <el-col :span="7"><el-button type="primary" style="width: 100%" :loading="loading" @click="load">筛选</el-button></el-col>
          </el-row>
          <el-table :data="datasets" size="small" v-loading="loading" empty-text="暂无已登记数据集">
            <el-table-column prop="name" label="名称" min-width="200" />
            <el-table-column label="来源" min-width="180">
              <template #default="scope">
                <div>{{ scope.row.sourceType === "remote" ? "服务器目录" : "本地目录" }}</div>
                <div class="muted-text">{{ scope.row.serverName || scope.row.serverId || "当前后端" }}</div>
              </template>
            </el-table-column>
            <el-table-column label="扫描摘要" min-width="190">
              <template #default="scope">
                <div>{{ scope.row.fileCount }} 文件 / {{ scope.row.directoryCount }} 目录</div>
                <div class="muted-text">{{ scope.row.size }} · {{ scope.row.detectedModality || scope.row.modality }}</div>
              </template>
            </el-table-column>
            <el-table-column label="索引" width="110">
              <template #default="scope"><StatusTag :status="scope.row.indexStatus" /></template>
            </el-table-column>
            <el-table-column label="最近扫描" width="170">
              <template #default="scope">{{ formatDateTime(scope.row.lastScanAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="220">
              <template #default="scope">
                <el-space>
                  <el-button text type="primary" @click="detail(scope.row.id)">详情</el-button>
                  <el-button
                    v-if="scope.row.sourceType === 'remote'"
                    text
                    type="success"
                    :loading="phase4WorkflowLoadingId === scope.row.id"
                    @click="openPhase4Workflow(scope.row)"
                  >
                    Phase4 Workflow
                  </el-button>
                  <el-tag v-else size="small" type="info">仅远程</el-tag>
                </el-space>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="section-space">
      <template #header>扫描候选结果</template>
      <el-table :data="scanResult?.candidates || []" size="small" empty-text="请先执行服务器扫描">
        <el-table-column prop="name" label="候选名称" min-width="180" />
        <el-table-column prop="path" label="目录路径" min-width="260" />
        <el-table-column label="规模" min-width="180">
          <template #default="scope">{{ scope.row.fileCount }} 文件 / {{ scope.row.directoryCount }} 目录</template>
        </el-table-column>
        <el-table-column label="大小" width="120">
          <template #default="scope">{{ scope.row.size || formatBytes(scope.row.totalSizeBytes) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="scope"><StatusTag :status="scope.row.status || 'none'" /></template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="scope">
            <el-button text type="primary" @click="openImportDialog(scope.row)">登记为数据集</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="importDialogVisible" title="登记数据集" width="720px">
      <el-form :model="importForm" label-width="120px">
        <el-form-item label="数据集名称" required>
          <el-input v-model="importForm.name" placeholder="例如：论文图文样本集" />
        </el-form-item>
        <el-form-item label="来源位置" required>
          <el-radio-group v-model="importForm.sourceType">
            <el-radio value="local">local</el-radio>
            <el-radio value="remote">remote</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="importForm.sourceType === 'remote'" label="服务器节点" required>
          <el-select v-model="importForm.serverId" placeholder="选择服务器" style="width: 100%">
            <el-option v-for="server in servers" :key="server.id" :label="`${server.name} (${server.host})`" :value="server.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="目录路径" required>
          <el-input v-model="importForm.path" placeholder="输入本地路径或远程目录路径">
            <template #append>
              <el-button :loading="validationLoading" @click="validatePath">校验路径</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item v-if="validationResult">
          <el-alert
            :type="validationResult.valid ? 'success' : 'error'"
            :title="validationResult.valid ? '路径校验通过' : '路径校验未通过'"
            :description="validationDescription"
            :closable="false"
            show-icon
          />
        </el-form-item>
        <el-form-item label="描述" required>
          <el-input v-model="importForm.description" type="textarea" :rows="3" placeholder="说明数据来源、用途和边界" />
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="importForm.tagsText" placeholder="多个标签用英文逗号分隔" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="模态类型">
              <el-select v-model="importForm.modality" placeholder="可留空，由扫描结果推断" clearable style="width: 100%">
                <el-option label="Text" value="text" />
                <el-option label="Image" value="image" />
                <el-option label="Audio" value="audio" />
                <el-option label="Video" value="video" />
                <el-option label="Multimodal" value="multimodal" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="版本">
              <el-input v-model="importForm.version" placeholder="默认 imported" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button @click="validatePath" :loading="validationLoading">重新校验</el-button>
        <el-button type="primary" :loading="importSubmitting" @click="submitImport">登记并生成扫描摘要</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { datasetApi, phase4Api, serverApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type {
  Dataset,
  Phase4DatasetProfile,
  DatasetPathValidationResult,
  ServerDatasetCandidate,
  ServerDatasetScanResult,
  ServerNode
} from "@/types/domain";
import { formatBytes, formatDateTime } from "@/utils/format";
import { ElMessage } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const datasets = ref<Dataset[]>([]);
const servers = ref<ServerNode[]>([]);
const scanResult = ref<ServerDatasetScanResult>();
const loading = ref(false);
const scanLoading = ref(false);
const validationLoading = ref(false);
const importSubmitting = ref(false);
const importDialogVisible = ref(false);
const validationResult = ref<DatasetPathValidationResult>();
const phase4WorkflowLoadingId = ref("");

const filters = reactive({
  keyword: "",
  sourceType: undefined as "local" | "remote" | undefined
});

const scanForm = reactive({
  serverId: "",
  rootPath: "",
  maxDepth: 2
});

const importForm = reactive({
  name: "",
  sourceType: "remote" as "local" | "remote",
  serverId: "",
  path: "",
  description: "",
  tagsText: "",
  modality: undefined as "text" | "image" | "audio" | "video" | "multimodal" | undefined,
  version: ""
});

const validationDescription = computed(() => {
  if (!validationResult.value) {
    return "";
  }
  const parts = [validationResult.value.message];
  if (validationResult.value.serverName) {
    parts.push(`节点：${validationResult.value.serverName}`);
  }
  parts.push(`模式：${validationResult.value.mode}`);
  parts.push(`检查时间：${formatDateTime(validationResult.value.checkedAt)}`);
  return parts.join(" | ");
});

watch(
  () => [importForm.sourceType, importForm.serverId, importForm.path],
  () => {
    validationResult.value = undefined;
  }
);

watch(
  () => importForm.sourceType,
  (sourceType) => {
    if (sourceType === "remote") {
      importForm.serverId = importForm.serverId || scanForm.serverId || servers.value[0]?.id || "";
      return;
    }
    importForm.serverId = "";
  },
  { immediate: true }
);

onMounted(async () => {
  await Promise.all([load(), loadServers()]);
});

async function load() {
  loading.value = true;
  try {
    datasets.value = await datasetApi.getDatasets(filters);
  } finally {
    loading.value = false;
  }
}

async function loadServers() {
  servers.value = await serverApi.getServers();
  if (!scanForm.serverId) {
    scanForm.serverId = servers.value[0]?.id || "";
  }
  if (importForm.sourceType === "remote" && !importForm.serverId) {
    importForm.serverId = scanForm.serverId;
  }
}

async function scanDatasets() {
  if (!scanForm.serverId) {
    ElMessage.warning("请先选择服务器节点");
    return;
  }
  scanLoading.value = true;
  try {
    scanResult.value = await datasetApi.scanServerDatasets({
      serverId: scanForm.serverId,
      rootPath: scanForm.rootPath.trim() || undefined,
      maxDepth: scanForm.maxDepth
    });
    ElMessage.success(`已发现 ${scanResult.value.candidates.length} 个候选数据集目录`);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    scanLoading.value = false;
  }
}

function detail(id: string) {
  router.push(`/datasets/${id}`);
}

async function openPhase4Workflow(dataset: Dataset) {
  if (dataset.sourceType !== "remote" || !dataset.serverId) {
    ElMessage.warning("Phase4 workflow 入口当前只支持已登记的远程数据集");
    return;
  }
  phase4WorkflowLoadingId.value = dataset.id;
  try {
    const profiles = await phase4Api.getPhase4DatasetProfiles({ status: "active" });
    let profile = findPhase4DatasetProfile(profiles, dataset);
    if (!profile) {
      profile = await phase4Api.createPhase4DatasetProfile({
        datasetName: dataset.name,
        taskType: "page_level_retrieval",
        modalityComposition: [dataset.detectedModality || dataset.modality || "multimodal"],
        fileStructureSnapshot: {
          sourceDatasetId: dataset.id,
          path: dataset.path
        },
        sampleStatistics: {
          fileCount: dataset.fileCount,
          directoryCount: dataset.directoryCount,
          totalSizeBytes: dataset.totalSizeBytes,
          detectedModality: dataset.detectedModality || dataset.modality
        },
        officialMetric: "Recall@5",
        officialBaseline: "Phase4 initial retrieval baseline",
        knownDifficulties: ["page-level retrieval first"],
        userNotes: dataset.description || `Created from dataset registration ${dataset.id}`,
        metadata: {
          sourceDatasetId: dataset.id,
          sourceDatasetName: dataset.name,
          tags: dataset.tags
        },
        sourceMode: "registered_path",
        serverId: dataset.serverId,
        serverPath: dataset.path,
        status: "active"
      });
    }
    const workflows = await phase4Api.getPhase4Workflows({ datasetProfileId: profile.id });
    const activeWorkflow = workflows.find((item) => item.status !== "archived");
    if (activeWorkflow) {
      await router.push(`/phase4/workflows/${activeWorkflow.id}`);
      return;
    }
    const detail = await phase4Api.createPhase4Workflow({
      datasetProfileId: profile.id,
      reader: {
        executionMode: "api"
      },
      idea: {
        executionMode: "api"
      },
      coding: {
        executionMode: "api",
        runnerMode: "shenzhenvlab",
        serverId: profile.serverId
      },
      writing: {
        executionMode: "api"
      }
    });
    ElMessage.success("已创建 Phase4 workflow");
    await router.push(`/phase4/workflows/${detail.workflow.id}`);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    phase4WorkflowLoadingId.value = "";
  }
}

function openImportDialog(candidate?: ServerDatasetCandidate) {
  importDialogVisible.value = true;
  if (candidate) {
    importForm.name = candidate.name;
    importForm.sourceType = "remote";
    importForm.serverId = scanResult.value?.serverId || scanForm.serverId;
    importForm.path = candidate.path;
    importForm.description = candidate.description || `从服务器扫描发现的目录：${candidate.path}`;
    importForm.modality = normalizeModality(candidate.modality);
  }
}

async function validatePath() {
  if (!importForm.path.trim()) {
    ElMessage.warning("请先填写数据集目录路径");
    return;
  }
  if (importForm.sourceType === "remote" && !importForm.serverId) {
    ElMessage.warning("远程数据集必须选择服务器节点");
    return;
  }
  validationLoading.value = true;
  try {
    validationResult.value = await datasetApi.validateDatasetPath({
      sourceType: importForm.sourceType,
      path: importForm.path.trim(),
      serverId: importForm.sourceType === "remote" ? importForm.serverId : undefined
    });
    if (validationResult.value.valid) {
      ElMessage.success("路径校验通过");
    } else {
      ElMessage.warning(validationResult.value.message);
    }
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    validationLoading.value = false;
  }
}

async function submitImport() {
  if (!importForm.name.trim() || !importForm.description.trim() || !importForm.path.trim()) {
    ElMessage.warning("请完整填写名称、路径和描述");
    return;
  }
  if (!validationResult.value?.valid) {
    ElMessage.warning("登记前请先完成路径校验，并确保校验通过");
    return;
  }
  importSubmitting.value = true;
  try {
    const created = await datasetApi.importDataset({
      name: importForm.name.trim(),
      sourceType: importForm.sourceType,
      path: importForm.path.trim(),
      description: importForm.description.trim(),
      tags: parseTags(importForm.tagsText),
      modality: importForm.modality,
      version: importForm.version.trim() || undefined,
      serverId: importForm.sourceType === "remote" ? importForm.serverId : undefined
    });
    ElMessage.success("数据集已登记，并生成了最新扫描摘要");
    resetImportForm();
    importDialogVisible.value = false;
    await load();
    router.push(`/datasets/${created.dataset.id}`);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    importSubmitting.value = false;
  }
}

function resetImportForm() {
  importForm.name = "";
  importForm.sourceType = "remote";
  importForm.serverId = scanForm.serverId;
  importForm.path = "";
  importForm.description = "";
  importForm.tagsText = "";
  importForm.modality = undefined;
  importForm.version = "";
  validationResult.value = undefined;
}

function parseTags(raw: string) {
  return raw
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeModality(value?: string) {
  const options = ["text", "image", "audio", "video", "multimodal"] as const;
  return options.find((item) => item === value) ?? undefined;
}

function findPhase4DatasetProfile(items: Phase4DatasetProfile[], dataset: Dataset) {
  return items.find((item) => item.serverId === dataset.serverId && item.serverPath === dataset.path);
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

.muted-text {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
</style>

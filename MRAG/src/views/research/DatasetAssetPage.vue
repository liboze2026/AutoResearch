<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">数据集资产</h1>
        <p class="page-subtitle">查看已注册的数据集资产，并把 MRAG 已扫描数据集提升为科研资产。</p>
      </div>
      <el-space wrap>
        <el-button @click="loadAll" :loading="loading">刷新</el-button>
        <el-button @click="registerDialogVisible = true">从扫描结果注册</el-button>
        <el-button type="primary" @click="createDialogVisible = true">手工创建</el-button>
      </el-space>
    </div>

    <el-card>
      <el-table :data="assets" v-loading="loading" size="small" empty-text="暂无数据集资产">
        <el-table-column prop="name" label="名称" min-width="220" />
        <el-table-column prop="taskType" label="任务类型" width="120" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="sourceType" label="来源" width="120" />
        <el-table-column prop="existingDatasetName" label="关联扫描数据集" min-width="180" />
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button text type="primary" @click="openDetail(scope.row.id)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="detailVisible" title="数据集资产详情" size="48%">
      <el-skeleton v-if="detailLoading" :rows="8" animated />
      <template v-else-if="detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="名称">{{ detail.asset.name }}</el-descriptions-item>
          <el-descriptions-item label="路径">{{ detail.asset.localOrRemotePath }}</el-descriptions-item>
          <el-descriptions-item label="描述">{{ detail.asset.descriptionMd || '-' }}</el-descriptions-item>
          <el-descriptions-item label="README">{{ detail.asset.readmeMd || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Loader">{{ detail.asset.loaderNoteMd || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Schema">{{ detail.asset.schemaNoteMd || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div class="subsection-title">来源关联</div>
        <el-table :data="detail.sources" size="small" empty-text="暂无来源关联">
          <el-table-column prop="existingDatasetName" label="扫描数据集" min-width="180" />
          <el-table-column prop="existingDatasetRef" label="数据集 ID" min-width="180" />
          <el-table-column prop="sourceKind" label="来源类型" width="120" />
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="registerDialogVisible" title="从扫描结果注册数据集资产" width="720px">
      <el-form label-width="130px">
        <el-form-item label="已扫描数据集">
          <el-select v-model="registerForm.existingDatasetRef" filterable style="width: 100%" placeholder="选择已存在的 MRAG 数据集">
            <el-option v-for="dataset in datasets" :key="dataset.id" :label="`${dataset.name} (${dataset.path})`" :value="dataset.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称覆盖"><el-input v-model="registerForm.name" /></el-form-item>
        <el-form-item label="任务类型"><el-input v-model="registerForm.taskType" placeholder="例如 text / image / retrieval" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="registerForm.descriptionMd" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="registerDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitRegister">注册</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="createDialogVisible" title="手工创建数据集资产" width="720px">
      <el-form :model="createForm" label-width="130px">
        <el-form-item label="名称"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item label="路径"><el-input v-model="createForm.localOrRemotePath" /></el-form-item>
        <el-form-item label="任务类型"><el-input v-model="createForm.taskType" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="createForm.descriptionMd" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="README"><el-input v-model="createForm.readmeMd" type="textarea" :rows="3" /></el-form-item>
        <el-form-item label="Loader"><el-input v-model="createForm.loaderNoteMd" type="textarea" :rows="2" /></el-form-item>
        <el-form-item label="Schema"><el-input v-model="createForm.schemaNoteMd" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { datasetApi, researchAssetApi } from "@/api";
import type { Dataset, DatasetAsset, DatasetAssetCreateRequest, DatasetAssetDetail, DatasetAssetRegisterFromScanRequest } from "@/types/domain";
import { ElMessage } from "element-plus";
import { onMounted, reactive, ref } from "vue";

const assets = ref<DatasetAsset[]>([]);
const datasets = ref<Dataset[]>([]);
const detail = ref<DatasetAssetDetail>();
const loading = ref(false);
const detailLoading = ref(false);
const detailVisible = ref(false);
const registerDialogVisible = ref(false);
const createDialogVisible = ref(false);
const submitting = ref(false);

const registerForm = reactive<DatasetAssetRegisterFromScanRequest>({
  existingDatasetRef: "",
  name: "",
  taskType: "",
  descriptionMd: ""
});

const createForm = reactive<DatasetAssetCreateRequest>({
  name: "",
  descriptionMd: "",
  taskType: "text",
  status: "active",
  sourceType: "manual",
  localOrRemotePath: "",
  readmeMd: "",
  loaderNoteMd: "",
  schemaNoteMd: ""
});

onMounted(() => {
  void loadAll();
});

async function loadAll() {
  loading.value = true;
  try {
    const [assetList, datasetList] = await Promise.all([researchAssetApi.getDatasetAssets(), datasetApi.getDatasets()]);
    assets.value = assetList;
    datasets.value = datasetList;
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    loading.value = false;
  }
}

async function openDetail(id: string) {
  detailVisible.value = true;
  detailLoading.value = true;
  try {
    detail.value = await researchAssetApi.getDatasetAssetById(id);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    detailLoading.value = false;
  }
}

async function submitRegister() {
  if (!registerForm.existingDatasetRef) {
    ElMessage.warning("请先选择已扫描数据集");
    return;
  }
  submitting.value = true;
  try {
    await researchAssetApi.registerDatasetAssetFromScan(registerForm);
    ElMessage.success("数据集资产已注册");
    registerDialogVisible.value = false;
    registerForm.existingDatasetRef = "";
    registerForm.name = "";
    registerForm.taskType = "";
    registerForm.descriptionMd = "";
    await loadAll();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
  }
}

async function submitCreate() {
  submitting.value = true;
  try {
    await researchAssetApi.createDatasetAsset(createForm);
    ElMessage.success("数据集资产已创建");
    createDialogVisible.value = false;
    createForm.name = "";
    createForm.descriptionMd = "";
    createForm.taskType = "text";
    createForm.status = "active";
    createForm.sourceType = "manual";
    createForm.localOrRemotePath = "";
    createForm.readmeMd = "";
    createForm.loaderNoteMd = "";
    createForm.schemaNoteMd = "";
    await loadAll();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
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

.subsection-title {
  margin: 16px 0 10px;
  font-weight: 600;
}
</style>
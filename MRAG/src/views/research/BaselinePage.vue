<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Baseline 管理</h1>
        <p class="page-subtitle">为 dataset asset 维护已知基线结果，供后续人工比较参考。</p>
      </div>
      <el-space wrap>
        <el-button @click="loadAll" :loading="loading">刷新</el-button>
        <el-button type="primary" @click="dialogVisible = true">创建 Baseline</el-button>
      </el-space>
    </div>

    <el-card>
      <el-table :data="baselines" v-loading="loading" size="small" empty-text="暂无 baseline">
        <el-table-column prop="name" label="名称" min-width="220" />
        <el-table-column prop="datasetAssetId" label="数据集资产" min-width="180" />
        <el-table-column prop="sourceType" label="来源" width="120" />
        <el-table-column label="操作" width="180">
          <template #default="scope">
            <el-space>
              <el-button text type="primary" @click="openDetail(scope.row.id)">详情</el-button>
              <el-button text @click="openEdit(scope.row.id)">编辑</el-button>
            </el-space>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="detailVisible" title="Baseline 详情" size="48%">
      <el-skeleton v-if="detailLoading" :rows="8" animated />
      <template v-else-if="detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="名称">{{ detail.baseline.name }}</el-descriptions-item>
          <el-descriptions-item label="数据集资产">{{ detail.datasetAsset.name }}</el-descriptions-item>
          <el-descriptions-item label="来源">{{ detail.baseline.sourceType }}</el-descriptions-item>
          <el-descriptions-item label="备注">{{ detail.baseline.noteMd || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div class="subsection-title">Metric Schema</div>
        <pre class="pre-block">{{ pretty(detail.baseline.metricSchemaJson) }}</pre>
        <div class="subsection-title">Result</div>
        <pre class="pre-block">{{ pretty(detail.baseline.resultJson) }}</pre>
      </template>
    </el-drawer>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑 Baseline' : '创建 Baseline'" width="760px">
      <el-form label-width="130px">
        <el-form-item label="数据集资产">
          <el-select v-model="form.datasetAssetId" filterable style="width: 100%">
            <el-option v-for="asset in assets" :key="asset.id" :label="asset.name" :value="asset.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="来源"><el-select v-model="form.sourceType" style="width: 100%"><el-option label="manual" value="manual" /><el-option label="result_archive" value="result_archive" /><el-option label="mixed" value="mixed" /></el-select></el-form-item>
        <el-form-item label="Metric Schema JSON"><el-input v-model="metricSchemaText" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="Result JSON"><el-input v-model="resultText" type="textarea" :rows="4" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="form.noteMd" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { researchAssetApi } from "@/api";
import type { Baseline, BaselineCreateRequest, BaselineDetail, DatasetAsset } from "@/types/domain";
import { ElMessage } from "element-plus";
import { onMounted, reactive, ref } from "vue";

const baselines = ref<Baseline[]>([]);
const assets = ref<DatasetAsset[]>([]);
const detail = ref<BaselineDetail>();
const loading = ref(false);
const detailLoading = ref(false);
const detailVisible = ref(false);
const dialogVisible = ref(false);
const submitting = ref(false);
const editingId = ref("");
const metricSchemaText = ref('{\n  "primary": "accuracy"\n}');
const resultText = ref('{\n  "accuracy": 0.8\n}');

const form = reactive<BaselineCreateRequest>({
  datasetAssetId: "",
  name: "",
  sourceType: "manual",
  noteMd: ""
});

onMounted(() => {
  void loadAll();
});

async function loadAll() {
  loading.value = true;
  try {
    const [baselineList, assetList] = await Promise.all([researchAssetApi.getBaselines(), researchAssetApi.getDatasetAssets()]);
    baselines.value = baselineList;
    assets.value = assetList;
    if (!form.datasetAssetId) {
      form.datasetAssetId = assetList[0]?.id || "";
    }
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
    detail.value = await researchAssetApi.getBaselineById(id);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    detailLoading.value = false;
  }
}

async function openEdit(id: string) {
  const result = await researchAssetApi.getBaselineById(id);
  editingId.value = id;
  form.datasetAssetId = result.baseline.datasetAssetId;
  form.name = result.baseline.name;
  form.sourceType = result.baseline.sourceType;
  form.noteMd = result.baseline.noteMd;
  metricSchemaText.value = JSON.stringify(result.baseline.metricSchemaJson || {}, null, 2);
  resultText.value = JSON.stringify(result.baseline.resultJson || {}, null, 2);
  dialogVisible.value = true;
}

async function submit() {
  submitting.value = true;
  try {
    const payload = {
      ...form,
      metricSchemaJson: JSON.parse(metricSchemaText.value || "{}"),
      resultJson: JSON.parse(resultText.value || "{}")
    };
    if (editingId.value) {
      await researchAssetApi.updateBaseline(editingId.value, payload);
      ElMessage.success("Baseline 已更新");
    } else {
      await researchAssetApi.createBaseline(payload);
      ElMessage.success("Baseline 已创建");
    }
    dialogVisible.value = false;
    resetForm();
    await loadAll();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
  }
}

function pretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}

function resetForm() {
  editingId.value = "";
  form.datasetAssetId = assets.value[0]?.id || "";
  form.name = "";
  form.sourceType = "manual";
  form.noteMd = "";
  metricSchemaText.value = '{\n  "primary": "accuracy"\n}';
  resultText.value = '{\n  "accuracy": 0.8\n}';
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

.pre-block {
  white-space: pre-wrap;
  background: var(--panel-alt);
  border: 1px solid var(--border);
  padding: 12px;
  border-radius: 8px;
}
</style>
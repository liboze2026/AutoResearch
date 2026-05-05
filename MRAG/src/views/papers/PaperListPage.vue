<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">论文资产</h1>
        <p class="page-subtitle">导入论文、查看状态，并触发解析与创新点抽取。</p>
      </div>
      <el-space wrap>
        <el-button @click="load" :loading="loading">刷新</el-button>
        <el-button type="primary" @click="dialogVisible = true">导入论文</el-button>
      </el-space>
    </div>

    <el-card>
      <el-table :data="papers" v-loading="loading" size="small" empty-text="暂无论文资产">
        <el-table-column prop="title" label="标题" min-width="220" />
        <el-table-column prop="authors" label="作者" min-width="180" />
        <el-table-column prop="status" label="状态" width="140" />
        <el-table-column prop="sourceType" label="来源" width="120" />
        <el-table-column label="更新时间" width="180">
          <template #default="scope">{{ formatDateTime(scope.row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="260">
          <template #default="scope">
            <el-space wrap>
              <el-button text type="primary" @click="openDetail(scope.row.id)">详情</el-button>
              <el-button text @click="runParse(scope.row.id)">解析</el-button>
              <el-button text @click="runExtract(scope.row.id)">抽取创新点</el-button>
            </el-space>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" title="导入论文" width="640px">
      <el-form label-width="120px">
        <el-form-item label="导入方式">
          <el-radio-group v-model="importMode">
            <el-radio value="file">本地文件上传</el-radio>
            <el-radio value="workspace">workspace 路径登记</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="importMode === 'file'" label="论文文件">
          <el-upload :auto-upload="false" :limit="1" :on-change="onFileChange" :show-file-list="true">
            <template #trigger>
              <el-button>选择文件</el-button>
            </template>
          </el-upload>
        </el-form-item>
        <el-form-item v-else label="workspace 路径">
          <el-input v-model="existingPath" placeholder="例如：/app/workspace/papers/incoming/demo.md 或 workspace/papers/incoming/demo.md" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitImport">导入</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { paperApi } from "@/api";
import type { Paper } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage, type UploadFile } from "element-plus";
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();
const papers = ref<Paper[]>([]);
const loading = ref(false);
const dialogVisible = ref(false);
const submitting = ref(false);
const importMode = ref<"file" | "workspace">("file");
const selectedFile = ref<File>();
const existingPath = ref("");

onMounted(() => {
  void load();
});

async function load() {
  loading.value = true;
  try {
    papers.value = await paperApi.getPapers();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    loading.value = false;
  }
}

function onFileChange(file: UploadFile) {
  selectedFile.value = file.raw;
}

async function submitImport() {
  submitting.value = true;
  try {
    if (importMode.value === "file") {
      if (!selectedFile.value) {
        ElMessage.warning("请先选择论文文件");
        return;
      }
      const result = await paperApi.importPaperFromFile(selectedFile.value);
      ElMessage.success(`论文已导入：${result.paper.title}`);
    } else {
      if (!existingPath.value.trim()) {
        ElMessage.warning("请填写 workspace 文件路径");
        return;
      }
      const result = await paperApi.importPaperFromWorkspace(existingPath.value.trim());
      ElMessage.success(`论文已登记：${result.paper.title}`);
    }
    dialogVisible.value = false;
    selectedFile.value = undefined;
    existingPath.value = "";
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
  }
}

function openDetail(id: string) {
  router.push(`/papers/${id}`);
}

async function runParse(id: string) {
  try {
    await paperApi.parsePaper(id);
    ElMessage.success("论文解析已触发");
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function runExtract(id: string) {
  try {
    await paperApi.extractPaperInsights(id);
    ElMessage.success("创新点抽取已完成");
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
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
</style>
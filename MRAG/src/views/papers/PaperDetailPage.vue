<template>
  <div>
    <el-page-header @back="$router.push('/papers')" content="论文详情" />

    <el-skeleton v-if="loading" :rows="10" animated class="section-space" />

    <div v-else-if="detail" class="section-space">
      <el-row :gutter="12">
        <el-col :span="14">
          <el-card>
            <template #header>
              <div class="card-header">
                <span>{{ detail.paper.title }}</span>
                <el-space>
                  <el-tag>{{ detail.paper.status }}</el-tag>
                  <el-button size="small" @click="runParse">重新解析</el-button>
                  <el-button size="small" type="primary" @click="runExtract">抽取创新点</el-button>
                </el-space>
              </div>
            </template>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="论文 ID">{{ detail.paper.id }}</el-descriptions-item>
              <el-descriptions-item label="来源">{{ detail.paper.sourceType }}</el-descriptions-item>
              <el-descriptions-item label="作者">{{ detail.paper.authors || '-' }}</el-descriptions-item>
              <el-descriptions-item label="年份">{{ detail.paper.year || '-' }}</el-descriptions-item>
              <el-descriptions-item label="Venue">{{ detail.paper.venue || '-' }}</el-descriptions-item>
              <el-descriptions-item label="解析模式">{{ detail.paper.parseMode || '-' }}</el-descriptions-item>
              <el-descriptions-item label="摘要" :span="2">{{ detail.paper.abstract || '暂无摘要' }}</el-descriptions-item>
              <el-descriptions-item label="解析备注" :span="2">{{ detail.paper.parserNote || detail.paper.parseError || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-card>

          <el-card class="section-space">
            <template #header>创新点与解析结果</template>
            <el-empty v-if="!detail.insightList?.length" description="当前还没有 insight，可先点击“抽取创新点”。" />
            <div v-else v-for="insight in detail.insightList" :key="insight.id" class="insight-block">
              <el-alert :title="`抽取状态：${insight.extractStatus}`" :type="insight.extractStatus === 'completed' ? 'success' : 'warning'" :closable="false" show-icon />
              <div class="subsection-title">Summary</div>
              <pre class="pre-block">{{ insight.summaryMd }}</pre>
              <div class="subsection-title">Contributions</div>
              <pre class="pre-block">{{ pretty(insight.contributionsJson) }}</pre>
              <div class="subsection-title">Methods</div>
              <pre class="pre-block">{{ pretty(insight.methodsJson) }}</pre>
              <div class="subsection-title">Limitations</div>
              <pre class="pre-block">{{ pretty(insight.limitationsJson) }}</pre>
            </div>
          </el-card>
        </el-col>

        <el-col :span="10">
          <el-card>
            <template #header>文件</template>
            <el-table :data="detail.files" size="small" empty-text="暂无文件">
              <el-table-column prop="fileType" label="类型" width="100" />
              <el-table-column prop="filePath" label="路径" min-width="220" show-overflow-tooltip />
            </el-table>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <el-empty v-else description="未找到论文详情" class="section-space" />
  </div>
</template>

<script setup lang="ts">
import { paperApi } from "@/api";
import type { PaperDetail } from "@/types/domain";
import { ElMessage } from "element-plus";
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
const detail = ref<PaperDetail>();
const loading = ref(false);

onMounted(() => {
  void loadDetail();
});

async function loadDetail() {
  loading.value = true;
  try {
    detail.value = await paperApi.getPaperById(route.params.id as string);
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    loading.value = false;
  }
}

async function runParse() {
  try {
    await paperApi.parsePaper(route.params.id as string);
    ElMessage.success("解析已完成");
    await loadDetail();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function runExtract() {
  try {
    await paperApi.extractPaperInsights(route.params.id as string);
    ElMessage.success("创新点抽取已完成");
    await loadDetail();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

function pretty(value: unknown) {
  return JSON.stringify(value ?? {}, null, 2);
}
</script>

<style scoped>
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.subsection-title {
  margin: 12px 0 8px;
  font-weight: 600;
}

.pre-block {
  white-space: pre-wrap;
  background: var(--panel-alt);
  border: 1px solid var(--border);
  padding: 12px;
  border-radius: 8px;
}

.insight-block + .insight-block {
  margin-top: 16px;
}
</style>
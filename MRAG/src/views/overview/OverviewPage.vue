<template>
  <div>
    <div class="page-header">
      <h1 class="page-title">系统总览</h1>
      <p class="page-subtitle">聚焦数据集资产、索引准备情况和服务器可用性，作为当前系统的统一入口。</p>
    </div>

    <div class="card-grid grid-4">
      <StatCard label="数据集数量" :value="stats?.datasetCount ?? '-'" hint="已登记的数据集总数" />
      <StatCard label="已扫描数据集" :value="stats?.scannedDatasets ?? '-'" hint="最近一次统计中已有扫描摘要的数据集" />
      <StatCard label="待构建索引" :value="stats?.pendingIndexes ?? '-'" hint="索引状态不是 ready 的数据集" />
      <StatCard label="可连接服务器" :value="stats ? `${stats.serverOnline} / ${stats.serverTotal}` : '-'" hint="最近一次状态刷新后在线的服务器节点" />
    </div>

    <el-row :gutter="12" class="section-space">
      <el-col :span="15">
        <el-card>
          <template #header>重点数据集</template>
          <el-table :data="displayDatasets" size="small" empty-text="暂无数据集">
            <el-table-column prop="name" label="名称" min-width="220" />
            <el-table-column label="来源" width="120">
              <template #default="scope">{{ scope.row.sourceType === "remote" ? "服务器" : "本地" }}</template>
            </el-table-column>
            <el-table-column label="规模" min-width="180">
              <template #default="scope">{{ scope.row.fileCount }} 文件 / {{ scope.row.directoryCount }} 目录</template>
            </el-table-column>
            <el-table-column label="索引" width="100">
              <template #default="scope"><StatusTag :status="scope.row.indexStatus" /></template>
            </el-table-column>
            <el-table-column label="最近扫描" width="180">
              <template #default="scope">{{ formatDateTime(scope.row.lastScanAt) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="100">
              <template #default="scope">
                <el-button text type="primary" @click="gotoDataset(scope.row.id)">详情</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="9">
        <el-card>
          <template #header>
            <div class="header-inline">
              <span>说明与快捷入口</span>
              <el-tag :type="stats?.statsMode === 'mock' ? 'warning' : 'success'">{{ stats?.statsMode === "mock" ? "演示统计" : "真实统计" }}</el-tag>
            </div>
          </template>
          <p class="intro-text">{{ stats?.platformIntro }}</p>
          <el-space wrap>
            <el-button type="primary" @click="$router.push('/datasets')">管理数据集</el-button>
            <el-button @click="$router.push('/servers')">管理服务器</el-button>
            <el-button @click="$router.push('/settings')">系统设置</el-button>
          </el-space>
          <el-divider />
          <p class="section-label">统计说明</p>
          <ul class="note-list">
            <li v-for="note in stats?.notes || []" :key="note">{{ note }}</li>
          </ul>
          <p class="section-label">服务器状态</p>
          <el-timeline>
            <el-timeline-item v-for="srv in servers" :key="srv.id" :timestamp="formatDateTime(srv.lastHeartbeat)" placement="top">
              {{ srv.name }} <StatusTag :status="srv.status" />
            </el-timeline-item>
          </el-timeline>
          <p class="meta-line">统计生成时间：{{ formatDateTime(stats?.statsGeneratedAt) }}</p>
        </el-card>
      </el-col>
    </el-row>

    <el-card class="section-space">
      <template #header>近 7 天平台趋势</template>
      <div ref="chartRef" style="height: 300px"></div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import StatCard from "@/components/StatCard.vue";
import StatusTag from "@/components/StatusTag.vue";
import { datasetApi, serverApi, systemApi } from "@/api";
import type { Dataset, OverviewStats, ServerNode } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { computed, nextTick, onMounted, onUnmounted, ref } from "vue";
import { useRouter } from "vue-router";
import * as echarts from "echarts";

const router = useRouter();
const stats = ref<OverviewStats>();
const datasets = ref<Dataset[]>([]);
const servers = ref<ServerNode[]>([]);
const chartRef = ref<HTMLDivElement>();
const displayDatasets = computed(() => datasets.value.slice(0, 6));
let chart: echarts.ECharts | null = null;

function gotoDataset(id: string) {
  router.push(`/datasets/${id}`);
}

function renderChart() {
  if (!chartRef.value || !stats.value) {
    return;
  }

  if (!chart) {
    chart = echarts.init(chartRef.value);
  }

  chart.setOption(
    {
      tooltip: { trigger: "axis" },
      legend: { top: 0 },
      xAxis: { type: "category", data: stats.value.trend.map((item) => item.date) },
      yAxis: { type: "value", minInterval: 1 },
      series: [
        {
          name: "数据集总数",
          type: "line",
          smooth: true,
          data: stats.value.trend.map((item) => item.datasets ?? 0)
        },
        {
          name: "已扫描数据集",
          type: "line",
          smooth: true,
          data: stats.value.trend.map((item) => item.scanned ?? 0)
        },
        {
          name: "在线服务器",
          type: "line",
          smooth: true,
          data: stats.value.trend.map((item) => item.onlineServers ?? 0)
        }
      ]
    },
    true
  );
}

onMounted(async () => {
  const [overviewStats, datasetList, serverList] = await Promise.all([
    systemApi.getOverviewStats(),
    datasetApi.getDatasets(),
    serverApi.getServers()
  ]);
  stats.value = overviewStats;
  datasets.value = datasetList;
  servers.value = serverList;
  await nextTick();
  renderChart();
});

onUnmounted(() => {
  chart?.dispose();
});
</script>

<style scoped>
.header-inline {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.intro-text {
  margin-top: 0;
}

.section-label {
  margin: 0 0 8px;
}

.note-list {
  padding-left: 18px;
  margin: 0 0 12px;
  color: var(--subtext);
}

.meta-line {
  margin: 0;
  color: var(--subtext);
}
</style>

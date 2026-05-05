<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">服务器管理</h1>
        <p class="page-subtitle">在现有服务器管理基础上，补充 heartbeat 和 GPU snapshot 展示，为阶段2调度器提供可视化资源视图。</p>
      </div>
      <el-space wrap>
        <el-button @click="load">刷新列表</el-button>
        <el-button type="primary" @click="openCreateDialog">新增服务器</el-button>
      </el-space>
    </div>

    <el-card>
      <template #header>服务器列表</template>
      <el-table :data="servers" size="small" v-loading="loading" empty-text="暂无服务器节点">
        <el-table-column prop="name" label="节点" width="150" />
        <el-table-column prop="host" label="目标地址" min-width="150" />
        <el-table-column prop="sshPort" label="端口" width="90" />
        <el-table-column prop="username" label="用户" width="110" />
        <el-table-column label="认证方式" width="120">
          <template #default="scope">{{ authTypeLabel(scope.row.authType) }}</template>
        </el-table-column>
        <el-table-column label="GPU" min-width="180">
          <template #default="scope">
            <div>{{ scope.row.gpuInfo || "未检测" }}</div>
            <div class="muted-text">{{ gpuSummary(scope.row) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="scope"><StatusTag :status="scope.row.status" /></template>
        </el-table-column>
        <el-table-column label="最近刷新" width="170">
          <template #default="scope">{{ formatDateTime(scope.row.lastHeartbeat) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="scope">
            <el-space wrap>
              <el-button text type="primary" @click="editServer(scope.row)">编辑</el-button>
              <el-button text @click="refreshStatus(scope.row.id)">状态</el-button>
              <el-button text @click="test(scope.row.id)">连接</el-button>
              <el-button text @click="checkGpu(scope.row.id)">GPU</el-button>
              <el-button text @click="collectHeartbeat(scope.row.id)">Heartbeat</el-button>
              <el-button text @click="collectGpuSnapshot(scope.row.id)">Snapshot</el-button>
              <el-button text @click="showResourceHistory(scope.row)">资源</el-button>
              <el-button text type="danger" @click="removeServer(scope.row.id)">删除</el-button>
            </el-space>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-row :gutter="12" class="section-space">
      <el-col :span="12">
        <el-card>
          <template #header>最近一次连接测试</template>
          <el-empty v-if="!lastProbe" description="点击“连接”后，这里会展示 SSH 登录验证结果与诊断信息。" />
          <el-descriptions v-else :column="1" border>
            <el-descriptions-item label="服务器">{{ lastProbe.serverName }}</el-descriptions-item>
            <el-descriptions-item label="测试模式">{{ lastProbe.mode === "mock" ? "演示模式" : "真实 SSH" }}</el-descriptions-item>
            <el-descriptions-item label="结果">{{ probeResultLabel(lastProbe.result) }}</el-descriptions-item>
            <el-descriptions-item label="说明">{{ lastProbe.message }}</el-descriptions-item>
            <el-descriptions-item label="SSH 目标">{{ lastProbe.target }}</el-descriptions-item>
            <el-descriptions-item label="远端主机">{{ lastProbe.remoteHost || "-" }}</el-descriptions-item>
            <el-descriptions-item label="远端用户">{{ lastProbe.remoteUser || "-" }}</el-descriptions-item>
            <el-descriptions-item label="耗时">{{ lastProbe.latencyMs }} ms</el-descriptions-item>
            <el-descriptions-item label="标准错误">{{ lastProbe.stderr || "-" }}</el-descriptions-item>
            <el-descriptions-item label="检查时间">{{ formatDateTime(lastProbe.checkedAt) }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>最近一次 GPU 检查</template>
          <el-empty v-if="!lastGpuProbe" description="点击“GPU”后，这里会展示空闲显卡检查结果。" />
          <template v-else>
            <el-alert
              :title="`${lastGpuProbe.serverName}：${lastGpuProbe.summary}`"
              :description="`可用 ${lastGpuProbe.availableGpuCount} / ${lastGpuProbe.totalGpuCount} 张 | 时间：${formatDateTime(lastGpuProbe.checkedAt)}`"
              type="info"
              :closable="false"
              show-icon
            />
            <el-table class="section-space" :data="lastGpuProbe.devices" size="small">
              <el-table-column prop="index" label="#" width="60" />
              <el-table-column prop="name" label="GPU" min-width="140" />
              <el-table-column label="显存" min-width="140">
                <template #default="scope">{{ memoryLabel(scope.row.memoryUsedMb, scope.row.memoryTotalMb) }}</template>
              </el-table-column>
              <el-table-column label="利用率" width="100">
                <template #default="scope">{{ scope.row.utilization ?? 0 }}%</template>
              </el-table-column>
              <el-table-column label="进程数" width="90">
                <template #default="scope">{{ scope.row.processes ?? 0 }}</template>
              </el-table-column>
              <el-table-column label="可用" width="90">
                <template #default="scope"><StatusTag :status="scope.row.available ? 'ready' : 'failed'" /></template>
              </el-table-column>
            </el-table>
          </template>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12" class="section-space">
      <el-col :span="12">
        <el-card>
          <template #header>Heartbeat 历史</template>
          <div class="section-inline header-inline">
            <strong>{{ resourceServerName || "未选择服务器" }}</strong>
            <el-button size="small" @click="refreshResourceHistory" :disabled="!resourceServerId">刷新历史</el-button>
          </div>
          <el-empty v-if="!resourceServerId" description="点击“资源”查看某台服务器的 heartbeat 历史。" />
          <el-table v-else :data="heartbeatHistory" size="small" empty-text="暂无 heartbeat 记录">
            <el-table-column label="状态" width="100">
              <template #default="{ row }"><StatusTag :status="row.status" /></template>
            </el-table-column>
            <el-table-column label="时间" width="170">
              <template #default="{ row }">{{ formatDateTime(row.heartbeatAt) }}</template>
            </el-table-column>
            <el-table-column label="摘要" min-width="220">
              <template #default="{ row }">{{ heartbeatSummary(row) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card>
          <template #header>GPU Snapshot 历史</template>
          <el-empty v-if="!resourceServerId" description="点击“资源”查看某台服务器的 GPU 快照。" />
          <el-table v-else :data="gpuSnapshots" size="small" empty-text="暂无 GPU 快照">
            <el-table-column prop="gpuIndex" label="#" width="60" />
            <el-table-column prop="name" label="GPU" min-width="140" />
            <el-table-column label="空闲显存" width="120">
              <template #default="{ row }">{{ row.freeMemMb }} MB</template>
            </el-table-column>
            <el-table-column label="利用率" width="100">
              <template #default="{ row }">{{ row.utilization }}%</template>
            </el-table-column>
            <el-table-column label="采集时间" width="170">
              <template #default="{ row }">{{ formatDateTime(row.capturedAt) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑服务器' : '新增服务器'" width="760px">
      <el-form :model="form" label-width="110px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="名称" required><el-input v-model="form.name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="主机/IP" required><el-input v-model="form.host" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="SSH 端口" required><el-input-number v-model="form.sshPort" :min="1" :max="65535" style="width: 100%" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="用户名" required><el-input v-model="form.username" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="认证方式" required>
              <el-select v-model="form.authType" style="width: 100%">
                <el-option label="SSH 配置" value="ssh_config" />
                <el-option label="密钥直连" value="key" />
                <el-option label="密码认证" value="password" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12"><el-form-item label="远程根目录" required><el-input v-model="form.remoteRoot" /></el-form-item></el-col>
        </el-row>
        <el-form-item v-if="form.authType === 'password'" label="登录密码" required>
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="editingId && form.hasPassword ? '已保存密码，如需修改请重新输入' : '请输入服务器登录密码'"
          />
        </el-form-item>
        <el-form-item label="任务目录" required><el-input v-model="form.taskWorkdir" /></el-form-item>
        <el-form-item label="Config JSON" required>
          <el-input v-model="form.configText" type="textarea" :rows="10" placeholder='例如：{"sshAlias":"node-a","workspace":"/data"}' />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitServer">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { experimentApi, serverApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { GPUProbeResult, GPUResourceSnapshot, ServerHeartbeat, ServerNode, ServerNodePayload, SSHConnectionTestResult } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { ElMessage, ElMessageBox } from "element-plus";
import { onMounted, reactive, ref } from "vue";

const servers = ref<ServerNode[]>([]);
const loading = ref(false);
const submitting = ref(false);
const dialogVisible = ref(false);
const editingId = ref<string>();
const lastProbe = ref<SSHConnectionTestResult>();
const lastGpuProbe = ref<GPUProbeResult>();
const resourceServerId = ref("");
const resourceServerName = ref("");
const heartbeatHistory = ref<ServerHeartbeat[]>([]);
const gpuSnapshots = ref<GPUResourceSnapshot[]>([]);

const form = reactive({
  name: "",
  host: "",
  sshPort: 22,
  username: "",
  authType: "ssh_config",
  password: "",
  hasPassword: false,
  remoteRoot: "",
  taskWorkdir: "",
  configText: "{}"
});

onMounted(() => {
  void load();
});

async function load() {
  loading.value = true;
  try {
    servers.value = await serverApi.getServers();
  } finally {
    loading.value = false;
  }
}

function authTypeLabel(authType: string) {
  if (authType === "ssh_config") {
    return "SSH 配置";
  }
  if (authType === "password") {
    return "密码认证";
  }
  if (authType === "key") {
    return "密钥直连";
  }
  return authType || "未设置";
}

function probeResultLabel(result: string) {
  const labels: Record<string, string> = {
    login_success: "登录成功",
    host_unreachable: "主机不可达",
    handshake_failed: "握手失败",
    auth_failed: "认证失败"
  };
  return labels[result] || result;
}

function gpuSummary(server: ServerNode) {
  if (server.availableGpus === undefined || server.totalGpus === undefined) {
    return server.lastGpuCheckAt ? `最近检查：${formatDateTime(server.lastGpuCheckAt)}` : "尚未检查空闲 GPU";
  }
  return `可用 ${server.availableGpus} / ${server.totalGpus} · ${formatDateTime(server.lastGpuCheckAt)}`;
}

function memoryLabel(used?: number, total?: number) {
  if (used === undefined || total === undefined) {
    return "-";
  }
  return `${used} / ${total} MB`;
}

function openCreateDialog() {
  editingId.value = undefined;
  resetForm();
  dialogVisible.value = true;
}

function editServer(server: ServerNode) {
  editingId.value = server.id;
  form.name = server.name;
  form.host = server.host;
  form.sshPort = server.sshPort;
  form.username = server.username;
  form.authType = server.authType;
  form.password = "";
  form.hasPassword = Boolean(server.hasPassword);
  form.remoteRoot = server.remoteRoot;
  form.taskWorkdir = server.taskWorkdir;
  form.configText = JSON.stringify(server.config || {}, null, 2);
  dialogVisible.value = true;
}

async function submitServer() {
  if (!form.name.trim() || !form.host.trim() || !form.username.trim() || !form.remoteRoot.trim() || !form.taskWorkdir.trim()) {
    ElMessage.warning("请完整填写服务器基本信息");
    return;
  }
  if (form.authType === "password" && !form.password.trim() && !(editingId.value && form.hasPassword)) {
    ElMessage.warning("密码认证方式下请填写登录密码");
    return;
  }

  let payload: ServerNodePayload;
  try {
    payload = {
      name: form.name.trim(),
      host: form.host.trim(),
      sshPort: form.sshPort,
      username: form.username.trim(),
      authType: form.authType,
      password: form.authType === "password" && form.password.trim() ? form.password : undefined,
      remoteRoot: form.remoteRoot.trim(),
      taskWorkdir: form.taskWorkdir.trim(),
      config: JSON.parse(form.configText || "{}")
    };
  } catch {
    ElMessage.error("Config JSON 不是合法的 JSON，请先修正");
    return;
  }

  submitting.value = true;
  try {
    if (editingId.value) {
      await serverApi.updateServer(editingId.value, payload);
      ElMessage.success("服务器配置已更新");
    } else {
      await serverApi.createServer(payload);
      ElMessage.success("服务器已创建");
    }
    dialogVisible.value = false;
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  } finally {
    submitting.value = false;
  }
}

async function removeServer(id: string) {
  try {
    await ElMessageBox.confirm("删除后将不再参与扫描和状态检查，是否继续？", "删除服务器", {
      type: "warning"
    });
    await serverApi.deleteServer(id);
    ElMessage.success("服务器已删除");
    await load();
  } catch (error) {
    if (error instanceof Error) {
      ElMessage.error(error.message);
    }
  }
}

async function test(id: string) {
  try {
    lastProbe.value = await serverApi.testServerConnection(id);
    if (lastProbe.value.reachable) {
      ElMessage.success("SSH 登录验证成功");
    } else {
      ElMessage.warning(`${probeResultLabel(lastProbe.value.result)}：${lastProbe.value.message}`);
    }
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function checkGpu(id: string) {
  try {
    lastGpuProbe.value = await serverApi.checkServerGpu(id);
    ElMessage.success(`GPU 检查完成，可用 ${lastGpuProbe.value.availableGpuCount} 张`);
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function collectHeartbeat(id: string) {
  try {
    const result = await experimentApi.triggerServerHeartbeat(id);
    resourceServerId.value = id;
    resourceServerName.value = servers.value.find((item) => item.id === id)?.name || id;
    ElMessage.success(`Heartbeat 完成：${result.heartbeat.status}`);
    await refreshResourceHistory();
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function collectGpuSnapshot(id: string) {
  try {
    const result = await experimentApi.triggerServerGpuSnapshot(id);
    resourceServerId.value = id;
    resourceServerName.value = servers.value.find((item) => item.id === id)?.name || id;
    ElMessage.success(`GPU Snapshot 完成，共 ${result.snapshots.length} 条`);
    await refreshResourceHistory();
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function showResourceHistory(server: ServerNode) {
  resourceServerId.value = server.id;
  resourceServerName.value = server.name;
  await refreshResourceHistory();
}

async function refreshResourceHistory() {
  if (!resourceServerId.value) {
    return;
  }
  try {
    const [heartbeats, snapshots] = await Promise.all([
      experimentApi.getServerHeartbeats(resourceServerId.value),
      experimentApi.getServerGpuSnapshots(resourceServerId.value)
    ]);
    heartbeatHistory.value = heartbeats;
    gpuSnapshots.value = snapshots;
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

async function refreshStatus(id: string) {
  try {
    const snapshot = await serverApi.refreshServerStatus(id);
    ElMessage.success(`状态已刷新：${snapshot.message}`);
    await load();
  } catch (error) {
    ElMessage.error((error as Error).message);
  }
}

function resetForm() {
  form.name = "";
  form.host = "";
  form.sshPort = 22;
  form.username = "";
  form.authType = "ssh_config";
  form.password = "";
  form.hasPassword = false;
  form.remoteRoot = "";
  form.taskWorkdir = "";
  form.configText = "{}";
}

function heartbeatSummary(item: ServerHeartbeat) {
  return String(item.detailJson?.message || item.detailJson?.result || "-");
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
  font-size: 12px;
}

.header-inline {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
</style>

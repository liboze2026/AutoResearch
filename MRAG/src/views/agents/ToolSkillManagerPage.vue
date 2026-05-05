<template>
  <div>
    <div class="page-header page-header-flex">
      <div>
        <h1 class="page-title">Tool / Skill Catalog</h1>
        <p class="page-subtitle">Inspect registered tools and skills, including paths, test state, dependencies, and reuse scope.</p>
      </div>
      <el-space wrap>
        <el-button :loading="loading" @click="loadAll">Refresh</el-button>
      </el-space>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="section-space" />

    <el-tabs>
      <el-tab-pane :label="`Tools (${tools.length})`">
        <el-card>
          <el-table :data="tools" v-loading="loading" size="small" empty-text="No tools">
            <el-table-column prop="name" label="Name" min-width="180" />
            <el-table-column prop="owner_agent_type" label="Owner" width="120" />
            <el-table-column prop="test_status" label="Test" width="120">
              <template #default="{ row }"><StatusTag :status="row.test_status" /></template>
            </el-table-column>
            <el-table-column prop="version" label="Version" width="100" />
            <el-table-column prop="path" label="Path" min-width="240" />
            <el-table-column prop="description" label="Description" min-width="220" />
          </el-table>
        </el-card>
      </el-tab-pane>
      <el-tab-pane :label="`Skills (${skills.length})`">
        <el-card>
          <el-table :data="skills" v-loading="loading" size="small" empty-text="No skills">
            <el-table-column prop="name" label="Name" min-width="180" />
            <el-table-column prop="entrypoint" label="Entrypoint" min-width="220" />
            <el-table-column prop="skill_dir" label="Skill Dir" min-width="220" />
            <el-table-column label="Dependencies" min-width="180">
              <template #default="{ row }">{{ row.dependencies.join(", ") || "-" }}</template>
            </el-table-column>
            <el-table-column prop="description" label="Description" min-width="220" />
          </el-table>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { agentApi } from "@/api";
import StatusTag from "@/components/StatusTag.vue";
import type { SkillDefinition, ToolDefinition } from "@/types/domain";
import { onMounted, ref } from "vue";

const loading = ref(false);
const error = ref("");
const tools = ref<ToolDefinition[]>([]);
const skills = ref<SkillDefinition[]>([]);

onMounted(() => {
  void loadAll();
});

async function loadAll() {
  loading.value = true;
  error.value = "";
  try {
    const [toolData, skillData] = await Promise.all([agentApi.getTools(), agentApi.getSkills()]);
    tools.value = toolData;
    skills.value = skillData;
  } catch (err) {
    error.value = (err as Error).message;
  } finally {
    loading.value = false;
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

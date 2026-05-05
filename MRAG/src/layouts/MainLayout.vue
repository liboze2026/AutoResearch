<template>
  <el-container class="layout-root">
    <el-aside :width="ui.collapsed ? '72px' : '230px'" class="layout-aside">
      <div class="brand">{{ ui.collapsed ? "MR" : "MRAG Platform" }}</div>
      <el-menu
        :default-active="$route.path"
        router
        :collapse="ui.collapsed"
        class="menu"
        background-color="#102341"
        text-color="#d3ddf0"
        active-text-color="#82a5ff"
      >
        <el-menu-item index="/overview">Overview</el-menu-item>
        <el-menu-item index="/agents">Agents</el-menu-item>
        <el-menu-item index="/agents/jobs">Agent Jobs</el-menu-item>
        <el-menu-item index="/agents/catalog">Tool / Skill</el-menu-item>
        <el-menu-item index="/agents/events">Pipeline Events</el-menu-item>
        <el-menu-item index="/datasets">Datasets</el-menu-item>
        <el-menu-item index="/phase4/workflows">Phase4 Workflow</el-menu-item>
        <el-menu-item index="/papers">Papers</el-menu-item>
        <el-menu-item index="/ideas">Ideas</el-menu-item>
        <el-menu-item index="/dataset-assets">Dataset Assets</el-menu-item>
        <el-menu-item index="/baselines">Baseline</el-menu-item>
        <el-menu-item index="/result-archives">Result Archives</el-menu-item>
        <el-menu-item index="/experiments">Experiments</el-menu-item>
        <el-menu-item index="/servers">Servers</el-menu-item>
        <el-menu-item index="/settings">Settings</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="layout-header">
        <el-button text @click="ui.toggleSidebar()">{{ ui.collapsed ? "Expand" : "Collapse" }}</el-button>
        <div class="header-right">
          <el-tag :type="runtimeProfile?.preset === 'all-real' ? 'success' : runtimeProfile?.preset === 'all-mock' ? 'warning' : 'info'">
            {{ presetLabel }}
          </el-tag>
          <el-tag v-for="item in runtimeProfile?.modes || []" :key="item.key" :type="item.mode === 'real' ? 'success' : 'warning'">
            {{ item.label }}: {{ item.mode === "real" ? "real" : "mock" }}
          </el-tag>
        </div>
      </el-header>
      <el-main class="layout-main">
        <el-alert
          v-if="runtimeProfile"
          class="mode-banner"
          :type="runtimeProfile.preset === 'all-real' ? 'success' : runtimeProfile.preset === 'all-mock' ? 'warning' : 'info'"
          :title="bannerTitle"
          :description="bannerDescription"
          :closable="false"
          show-icon
        />
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { systemApi } from "@/api";
import { useUiStore } from "@/store/ui";
import type { RuntimeProfile } from "@/types/domain";
import { computed, onMounted, ref } from "vue";

const ui = useUiStore();
const runtimeProfile = ref<RuntimeProfile>();

const presetLabel = computed(() => {
  switch (runtimeProfile.value?.preset) {
    case "all-real":
      return "Mode: all real";
    case "all-mock":
      return "Mode: all mock";
    case "mixed":
      return "Mode: mixed";
    default:
      return "Mode: loading";
  }
});

const bannerTitle = computed(() => {
  if (!runtimeProfile.value) {
    return "";
  }
  if (runtimeProfile.value.preset === "all-real") {
    return "The platform is running major capabilities in real mode.";
  }
  if (runtimeProfile.value.preset === "all-mock") {
    return "The platform is running in mock mode for demos and safe validation.";
  }
  return "The platform is in mixed mode and capabilities may run in mock or real paths.";
});

const bannerDescription = computed(() => {
  if (!runtimeProfile.value) {
    return "";
  }
  return runtimeProfile.value.modes.map((item) => `${item.label}: ${item.summary}`).join(" | ");
});

onMounted(async () => {
  runtimeProfile.value = await systemApi.getRuntimeProfile();
});
</script>

<style scoped>
.layout-root {
  min-height: 100vh;
}

.layout-aside {
  background: linear-gradient(180deg, #0f2344 0%, #0a1a33 100%);
  transition: width 0.2s ease;
}

.brand {
  height: 56px;
  color: #e8efff;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid #20365e;
}

.menu {
  border-right: none;
}

.layout-header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border);
  background: rgba(255, 255, 255, 0.7);
  backdrop-filter: blur(4px);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.layout-main {
  padding: 18px;
}

.mode-banner {
  margin-bottom: 18px;
}
</style>

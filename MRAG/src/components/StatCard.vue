<template>
  <el-card shadow="hover" class="stat-card">
    <div class="label">{{ label }}</div>
    <div class="value">{{ value }}</div>
    <div class="hint">{{ hint }}</div>
  </el-card>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  value: string | number;
  hint: string;
}>();
</script>

<style scoped>
.stat-card .label {
  color: var(--subtext);
  font-size: 13px;
}

.stat-card .value {
  font-size: 28px;
  margin-top: 8px;
  font-weight: 600;
}

.stat-card .hint {
  margin-top: 8px;
  color: var(--subtext);
  font-size: 12px;
}
</style>

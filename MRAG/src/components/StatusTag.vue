<template>
  <el-tag :type="meta.type" effect="light">{{ meta.text }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{ status: string }>();

const meta = computed(() => {
  const map: Record<string, { text: string; type: "success" | "warning" | "danger" | "info" | "primary" }> = {
    none: { text: "\u672A\u6784\u5EFA", type: "info" },
    registered: { text: "\u5DF2\u6CE8\u518C", type: "info" },
    idle: { text: "\u7A7A\u95F2", type: "info" },
    waiting_input: { text: "\u7B49\u5F85\u8F93\u5165", type: "warning" },
    ready: { text: "\u5C31\u7EEA", type: "success" },
    building: { text: "\u6784\u5EFA\u4E2D", type: "warning" },
    failed: { text: "\u5931\u8D25", type: "danger" },
    running: { text: "\u8FD0\u884C\u4E2D", type: "primary" },
    validating: { text: "\u6821\u9A8C\u4E2D", type: "warning" },
    repairing: { text: "\u4FEE\u590D\u4E2D", type: "warning" },
    preparing: { text: "\u51C6\u5907\u4E2D", type: "warning" },
    scheduled: { text: "\u5DF2\u8C03\u5EA6", type: "primary" },
    completed: { text: "\u5DF2\u5B8C\u6210", type: "success" },
    passed: { text: "\u5DF2\u901A\u8FC7", type: "success" },
    not_needed: { text: "\u65E0\u9700", type: "info" },
    succeeded: { text: "\u6210\u529F", type: "success" },
    queued: { text: "\u6392\u961F\u4E2D", type: "info" },
    stopped: { text: "\u5DF2\u505C\u6B62", type: "warning" },
    paused: { text: "\u5DF2\u6682\u505C", type: "warning" },
    draft: { text: "\u8349\u7A3F", type: "info" },
    processed: { text: "\u5DF2\u5904\u7406", type: "success" },
    spec_ready: { text: "Spec \u5C31\u7EEA", type: "primary" },
    archived: { text: "\u5DF2\u5F52\u6863", type: "info" },
    cancelled: { text: "\u5DF2\u53D6\u6D88", type: "warning" },
    online: { text: "\u5728\u7EBF", type: "success" },
    offline: { text: "\u79BB\u7EBF", type: "danger" },
    connecting: { text: "\u8FDE\u63A5\u4E2D", type: "warning" }
  };
  return map[props.status] || { text: props.status, type: "info" };
});
</script>

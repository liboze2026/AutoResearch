<template>
  <div class="settings-page">
    <div class="page-header">
      <h1 class="page-title">&#31995;&#32479;&#35774;&#32622;</h1>
      <p class="page-subtitle">&#36825;&#37324;&#19981;&#20877;&#32500;&#25252;&#26080;&#23454;&#38469;&#20316;&#29992;&#30340;&#34920;&#21333;&#37197;&#32622;&#65292;&#21482;&#20445;&#30041;&#24403;&#21069;&#31995;&#32479;&#36816;&#34892;&#26041;&#24335;&#21644;&#20351;&#29992;&#35828;&#26126;&#12290;</p>
    </div>

    <el-row :gutter="12" class="section-space" v-if="runtimeProfile">
      <el-col :span="15">
        <el-card class="panel-card">
          <template #header>&#24403;&#21069;&#36816;&#34892;&#27169;&#24335;</template>
          <el-table :data="runtimeProfile.modes" size="small">
            <el-table-column prop="label" label="&#33021;&#21147;" width="160" />
            <el-table-column label="&#24403;&#21069;&#27169;&#24335;" width="120">
              <template #default="scope">
                <el-tag :type="scope.row.mode === 'real' ? 'success' : 'warning'">{{ scope.row.mode }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="summary" label="&#24403;&#21069;&#34892;&#20026;&#35828;&#26126;" min-width="260" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="9">
        <el-card class="panel-card">
          <template #header>&#31995;&#32479;&#27010;&#35272;</template>
          <p class="meta-line">&#36816;&#34892;&#24418;&#24577;&#65306;{{ presetLabel }}</p>
          <p class="meta-line">&#26381;&#21153;&#22120;&#36830;&#25509;&#27169;&#24335;&#65306;{{ runtimeProfile.serverConnectionMode }}</p>
          <p class="meta-line">&#29983;&#25104;&#26102;&#38388;&#65306;{{ formatDateTime(runtimeProfile.generatedAt) }}</p>
          <el-divider />
          <ul class="note-list">
            <li v-for="note in runtimeProfile.notes" :key="note">{{ note }}</li>
          </ul>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="12" class="section-space">
      <el-col :span="12">
        <el-card class="panel-card">
          <template #header>&#24403;&#21069;&#39029;&#38754;&#20445;&#30041;&#20869;&#23481;</template>
          <ul class="note-list compact">
            <li>&#23637;&#31034;&#21518;&#31471;&#30495;&#23454;&#36816;&#34892;&#27169;&#24335;&#65292;&#26041;&#20415;&#30830;&#35748;&#24403;&#21069;&#26159; real &#36824;&#26159; mock&#12290;</li>
            <li>&#24110;&#21161;&#23450;&#20301;&#20026;&#20160;&#20040;&#26381;&#21153;&#22120;&#36830;&#25509;&#12289;&#25968;&#25454;&#38598;&#25195;&#25551;&#12289;&#32034;&#24341;&#26500;&#24314;&#34920;&#29616;&#19981;&#21516;&#12290;</li>
            <li>&#20316;&#20026;&#31995;&#32479;&#29366;&#24577;&#35828;&#26126;&#39029;&#65292;&#19981;&#20877;&#25215;&#25285;&#8220;&#37197;&#32622;&#20013;&#24515;&#8221;&#30340;&#32844;&#36131;&#12290;</li>
          </ul>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card class="panel-card">
          <template #header>&#37197;&#32622;&#20837;&#21475;&#35828;&#26126;</template>
          <ul class="note-list compact">
            <li>&#26381;&#21153;&#22120;&#36830;&#25509;&#21442;&#25968;&#35831;&#22312; <code>/servers</code> &#39029;&#38754;&#32500;&#25252;&#12290;</li>
            <li>&#25968;&#25454;&#38598;&#26469;&#28304;&#19982;&#25195;&#25551;&#20837;&#21475;&#35831;&#22312; <code>/datasets</code> &#39029;&#38754;&#23436;&#25104;&#12290;</li>
            <li>&#36816;&#34892;&#27169;&#24335;&#20999;&#25442;&#35831;&#20462;&#25913;&#21518;&#31471;&#29615;&#22659;&#21464;&#37327;&#25110; Docker Compose&#12290;</li>
          </ul>
          <p class="meta-tip">&#36816;&#34892;&#27169;&#24335;&#35828;&#26126;&#25991;&#26723;&#65306;<code>docs/runtime-modes-and-acceptance.md</code></p>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { systemApi } from "@/api";
import type { RuntimeProfile } from "@/types/domain";
import { formatDateTime } from "@/utils/format";
import { computed, onMounted, ref } from "vue";

const runtimeProfile = ref<RuntimeProfile>();

const presetLabel = computed(() => {
  switch (runtimeProfile.value?.preset) {
    case "all-real":
      return "\u5168 real";
    case "all-mock":
      return "\u5168 mock";
    case "mixed":
      return "\u6df7\u5408\u6a21\u5f0f";
    default:
      return "\u672a\u52a0\u8f7d";
  }
});

onMounted(async () => {
  runtimeProfile.value = await systemApi.getRuntimeProfile();
});
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.panel-card {
  min-height: 100%;
}

.meta-line {
  margin: 0 0 8px;
}

.meta-tip {
  margin: 12px 0 0;
  color: var(--el-text-color-secondary);
}

.note-list {
  margin: 0;
  padding-left: 18px;
  color: var(--el-text-color-regular);
}

.note-list.compact li + li {
  margin-top: 8px;
}
</style>
import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";
import MainLayout from "@/layouts/MainLayout.vue";

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    component: MainLayout,
    redirect: "/overview",
    children: [
      {
        path: "overview",
        name: "overview",
        component: () => import("@/views/overview/OverviewPage.vue")
      },
      {
        path: "agents",
        name: "agent-list",
        component: () => import("@/views/agents/AgentListPage.vue")
      },
      {
        path: "agents/jobs",
        name: "agent-job-list",
        component: () => import("@/views/agents/AgentJobListPage.vue")
      },
      {
        path: "agents/jobs/:id",
        name: "agent-job-detail",
        component: () => import("@/views/agents/AgentJobDetailPage.vue")
      },
      {
        path: "agents/catalog",
        name: "agent-catalog",
        component: () => import("@/views/agents/ToolSkillManagerPage.vue")
      },
      {
        path: "agents/events",
        name: "agent-events",
        component: () => import("@/views/agents/AgentEventPage.vue")
      },
      {
        path: "datasets",
        name: "dataset-list",
        component: () => import("@/views/datasets/DatasetListPage.vue")
      },
      {
        path: "phase4/workflows",
        name: "phase4-workflow-list",
        component: () => import("@/views/phase4/Phase4WorkflowListPage.vue")
      },
      {
        path: "phase4/workflows/:id",
        name: "phase4-workflow-detail",
        component: () => import("@/views/phase4/Phase4WorkflowDetailPage.vue")
      },
      {
        path: "datasets/:id",
        name: "dataset-detail",
        component: () => import("@/views/datasets/DatasetDetailPage.vue")
      },
      {
        path: "papers",
        name: "paper-list",
        component: () => import("@/views/papers/PaperListPage.vue")
      },
      {
        path: "papers/:id",
        name: "paper-detail",
        component: () => import("@/views/papers/PaperDetailPage.vue")
      },
      {
        path: "ideas",
        name: "idea-pool",
        component: () => import("@/views/ideas/IdeaPoolPage.vue")
      },
      {
        path: "dataset-assets",
        name: "dataset-assets",
        component: () => import("@/views/research/DatasetAssetPage.vue")
      },
      {
        path: "baselines",
        name: "baseline-page",
        component: () => import("@/views/research/BaselinePage.vue")
      },
      {
        path: "result-archives",
        name: "result-archives",
        component: () => import("@/views/research/ResultArchivePage.vue")
      },
      {
        path: "experiments",
        name: "experiment-list",
        component: () => import("@/views/experiments/ExperimentListPage.vue")
      },
      {
        path: "experiments/:id",
        name: "experiment-detail",
        component: () => import("@/views/experiments/ExperimentDetailPage.vue")
      },
      {
        path: "experiments/:id/comparisons",
        name: "experiment-comparisons",
        component: () => import("@/views/experiments/ExperimentComparePage.vue")
      },
      {
        path: "servers",
        name: "server-manage",
        component: () => import("@/views/servers/ServerPage.vue")
      },
      {
        path: "settings",
        name: "system-settings",
        component: () => import("@/views/settings/SettingsPage.vue")
      }
    ]
  }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

export default router;

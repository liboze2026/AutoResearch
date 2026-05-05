import { http } from "@/api/http";
import type { OverviewStats, RuntimeProfile } from "@/types/domain";

export async function getOverviewStats(): Promise<OverviewStats> {
  return http<OverviewStats>("/overview/stats");
}

export async function getRuntimeProfile(): Promise<RuntimeProfile> {
  return http<RuntimeProfile>("/runtime/profile");
}
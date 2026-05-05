import { http } from "@/api/http";
import type { Idea, IdeaCreateRequest, IdeaDetail, IdeaGenerationResult, IdeaUpdateRequest } from "@/types/domain";

export function getIdeas(): Promise<Idea[]> {
  return http<Idea[]>("/ideas");
}

export function getIdeaById(id: string): Promise<IdeaDetail> {
  return http<IdeaDetail>(`/ideas/${id}`);
}

export function createIdea(payload: IdeaCreateRequest): Promise<IdeaDetail> {
  return http<IdeaDetail>("/ideas", {
    method: "POST",
    body: payload
  });
}

export function updateIdea(id: string, payload: IdeaUpdateRequest): Promise<IdeaDetail> {
  return http<IdeaDetail>(`/ideas/${id}`, {
    method: "PATCH",
    body: payload
  });
}

export function generateIdeasFromPaper(paperId: string): Promise<IdeaGenerationResult> {
  return http<IdeaGenerationResult>(`/ideas/generate-from-paper/${paperId}`, {
    method: "POST",
    body: {}
  });
}
// Client-side API wrapper — talks to the Go backend over REST/JSON.
//
// The session lives in an HttpOnly cookie set by the backend, so every request
// goes out with `credentials: "include"`. Set NEXT_PUBLIC_API_BASE_URL to the
// API origin (defaults to the local dev server).

import type {
  Channel,
  ChannelInput,
  Generation,
  Profile,
  ProfileInput,
  PromptTemplate,
  PromptTemplateInput,
} from "./types";

export const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  status: number;
  /** Per-field messages from a 422, keyed by input field name. */
  fieldErrors?: Record<string, string>;

  constructor(
    message: string,
    status: number,
    fieldErrors?: ApiError["fieldErrors"],
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fieldErrors = fieldErrors;
  }
}

/** Low-level JSON request. Exported so lib/auth.ts shares the same behaviour. */
export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    ...init,
  });

  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => ({}));

  if (!res.ok) {
    // A 401 on a normal request means the session lapsed mid-use. Tell the
    // app so it can bounce to /login. Login/logout 401s are the caller's to
    // handle, so skip those.
    if (
      res.status === 401 &&
      !path.startsWith("/auth/") &&
      typeof window !== "undefined"
    ) {
      window.dispatchEvent(new Event("auth:expired"));
    }
    throw new ApiError(
      data.error ?? `Request failed (${res.status})`,
      res.status,
      data.errors,
    );
  }
  return data as T;
}

export function listProfiles() {
  return request<{ profiles: Profile[] }>("/profiles").then((r) => r.profiles);
}

export function getProfile(id: string) {
  return request<{ profile: Profile }>(`/profiles/${id}`).then((r) => r.profile);
}

export function createProfile(input: ProfileInput) {
  return request<{ profile: Profile }>("/profiles", {
    method: "POST",
    body: JSON.stringify(input),
  }).then((r) => r.profile);
}

export function updateProfile(id: string, input: ProfileInput) {
  return request<{ profile: Profile }>(`/profiles/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.profile);
}

export function deleteProfile(id: string) {
  return request<void>(`/profiles/${id}`, { method: "DELETE" });
}

// --- Channels (nested under a profile) ---

export function listChannels(profileId: string) {
  return request<{ channels: Channel[] }>(
    `/channels?profileId=${encodeURIComponent(profileId)}`,
  ).then((r) => r.channels);
}

export function getChannel(id: string) {
  return request<{ channel: Channel }>(`/channels/${id}`).then((r) => r.channel);
}

export function createChannel(profileId: string, input: ChannelInput) {
  return request<{ channel: Channel }>("/channels", {
    method: "POST",
    body: JSON.stringify({ profileId, ...input }),
  }).then((r) => r.channel);
}

export function updateChannel(id: string, input: ChannelInput) {
  return request<{ channel: Channel }>(`/channels/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.channel);
}

export function deleteChannel(id: string) {
  return request<void>(`/channels/${id}`, { method: "DELETE" });
}

// --- Prompt templates (nested under a profile) ---

export function listPromptTemplates(profileId: string) {
  return request<{ promptTemplates: PromptTemplate[] }>(
    `/prompt-templates?profileId=${encodeURIComponent(profileId)}`,
  ).then((r) => r.promptTemplates);
}

export function getPromptTemplate(id: string) {
  return request<{ promptTemplate: PromptTemplate }>(
    `/prompt-templates/${id}`,
  ).then((r) => r.promptTemplate);
}

export function createPromptTemplate(
  profileId: string,
  input: PromptTemplateInput,
) {
  return request<{ promptTemplate: PromptTemplate }>("/prompt-templates", {
    method: "POST",
    body: JSON.stringify({ profileId, ...input }),
  }).then((r) => r.promptTemplate);
}

export function updatePromptTemplate(id: string, input: PromptTemplateInput) {
  return request<{ promptTemplate: PromptTemplate }>(`/prompt-templates/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.promptTemplate);
}

export function deletePromptTemplate(id: string) {
  return request<void>(`/prompt-templates/${id}`, { method: "DELETE" });
}

// --- Generations (running a prompt template through the AI provider) ---

export function listGenerations(profileId: string) {
  return request<{ generations: Generation[] }>(
    `/generations?profileId=${encodeURIComponent(profileId)}`,
  ).then((r) => r.generations);
}

export function getGeneration(id: string) {
  return request<{ generation: Generation }>(`/generations/${id}`).then(
    (r) => r.generation,
  );
}

export function runGeneration(promptTemplateId: string) {
  return request<{ generation: Generation }>("/generations", {
    method: "POST",
    body: JSON.stringify({ promptTemplateId }),
  }).then((r) => r.generation);
}

export function deleteGeneration(id: string) {
  return request<void>(`/generations/${id}`, { method: "DELETE" });
}

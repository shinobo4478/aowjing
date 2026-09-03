// Client-side API wrapper.
//
// Today these hit the Next.js route handlers under app/api/ (backed by the
// temporary in-memory store). When the Go backend lands, set
// NEXT_PUBLIC_API_BASE_URL to its origin (e.g. http://localhost:8080) and
// adjust the paths to match the Go routes — the call sites in the pages
// shouldn't need to change.

import type { Profile, ProfileInput } from "./types";

const BASE = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

export class ApiError extends Error {
  status: number;
  fieldErrors?: Partial<Record<keyof ProfileInput, string>>;

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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });

  if (res.status === 204) return undefined as T;

  const data = await res.json().catch(() => ({}));

  if (!res.ok) {
    throw new ApiError(
      data.error ?? `Request failed (${res.status})`,
      res.status,
      data.errors,
    );
  }
  return data as T;
}

export function listProfiles() {
  return request<{ profiles: Profile[] }>("/api/profiles").then((r) => r.profiles);
}

export function getProfile(id: string) {
  return request<{ profile: Profile }>(`/api/profiles/${id}`).then((r) => r.profile);
}

export function createProfile(input: ProfileInput) {
  return request<{ profile: Profile }>("/api/profiles", {
    method: "POST",
    body: JSON.stringify(input),
  }).then((r) => r.profile);
}

export function updateProfile(id: string, input: ProfileInput) {
  return request<{ profile: Profile }>(`/api/profiles/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.profile);
}

export function deleteProfile(id: string) {
  return request<void>(`/api/profiles/${id}`, { method: "DELETE" });
}

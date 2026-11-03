// Client-side API wrapper — talks to the Go backend over REST/JSON.
//
// The session lives in an HttpOnly cookie set by the backend, so every request
// goes out with `credentials: "include"`. Set NEXT_PUBLIC_API_BASE_URL to the
// API origin (defaults to the local dev server).

import type {
  Channel,
  ChannelInput,
  Generation,
  Placement,
  Product,
  ProductInput,
  ProductUsage,
  ProductWithUse,
  Profile,
  ProfileInput,
  PromptTemplate,
  PromptTemplateInput,
  Settings,
  YouTubeAccount,
  YouTubePrivacy,
  YouTubeStatus,
  YouTubeUpload,
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

/** `productIds` (0–2, in pin order) fill the template's {{product.*}}
 *  variables and are pinned to the new clip. */
export function runGeneration(promptTemplateId: string, productIds: string[] = []) {
  return request<{ generation: Generation }>("/generations", {
    method: "POST",
    body: JSON.stringify({ promptTemplateId, productIds }),
  }).then((r) => r.generation);
}

export function deleteGeneration(id: string) {
  return request<void>(`/generations/${id}`, { method: "DELETE" });
}

// --- Settings (global provider credentials) ---

export function getSettings() {
  return request<{ settings: Settings }>("/settings").then((r) => r.settings);
}

/** Upserts only the keys present in `partial`; returns the full settings. */
export function updateSettings(partial: Partial<Settings>) {
  return request<{ settings: Settings }>("/settings", {
    method: "PUT",
    body: JSON.stringify(partial),
  }).then((r) => r.settings);
}

// --- YouTube (connect a channel, publish a video generation) ---

/** Connection state for a single channel. */
export function getYouTubeStatus(channelId: string) {
  return request<YouTubeStatus>(
    `/youtube/status?channelId=${encodeURIComponent(channelId)}`,
  );
}

/** Every connected account under a profile, plus whether the server has an
 *  OAuth client at all — one request for the whole channels table. */
export function listYouTubeAccounts(profileId: string) {
  return request<{ configured: boolean; accounts: YouTubeAccount[] }>(
    `/youtube/accounts?profileId=${encodeURIComponent(profileId)}`,
  );
}

/** URL for the OAuth handshake. Assign it to window.location — the callback
 *  needs a top-level navigation, not a fetch. */
export function youtubeConnectUrl(channelId: string) {
  return `${API_BASE}/youtube/oauth/start?channelId=${encodeURIComponent(channelId)}`;
}

export function disconnectYouTube(channelId: string) {
  return request<void>(`/youtube/accounts/${channelId}`, { method: "DELETE" });
}

export function listYouTubeUploads(profileId: string) {
  return request<{ uploads: YouTubeUpload[] }>(
    `/youtube/uploads?profileId=${encodeURIComponent(profileId)}`,
  ).then((r) => r.uploads);
}

export function publishToYouTube(input: {
  generationId: string;
  channelId: string;
  title: string;
  description: string;
  privacy: YouTubePrivacy;
}) {
  return request<{ upload?: YouTubeUpload; videoId?: string; url?: string }>(
    "/youtube/publish",
    { method: "POST", body: JSON.stringify(input) },
  );
}

// --- Products (central Shopee catalog) ---

export function listProducts(q = "") {
  const term = q.trim();
  const qs = term ? `?q=${encodeURIComponent(term)}` : "";
  return request<{ products: ProductWithUse[] }>(`/products${qs}`).then(
    (r) => r.products,
  );
}

export function createProduct(input: ProductInput) {
  return request<{ product: Product }>("/products", {
    method: "POST",
    body: JSON.stringify(input),
  }).then((r) => r.product);
}

export function updateProduct(id: string, input: ProductInput) {
  return request<{ product: Product }>(`/products/${id}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.product);
}

/** The clips a product is pinned to, newest first. */
export function getProductUsage(id: string) {
  return request<{ clips: ProductUsage[] }>(`/products/${id}/usage`).then(
    (r) => r.clips,
  );
}

export function deleteProduct(id: string) {
  return request<void>(`/products/${id}`, { method: "DELETE" });
}

// --- Placements (products pinned to a clip) ---

/** Every pin under a profile, so the generations table needs one request. */
export function listPlacements(profileId: string) {
  return request<{ placements: Placement[] }>(
    `/placements?profileId=${encodeURIComponent(profileId)}`,
  ).then((r) => r.placements);
}

/** Replaces a clip's pinned products (0–2 ids; order = pin order). */
export function setPlacements(generationId: string, productIds: string[]) {
  return request<{ placements: Placement[] }>(`/placements/${generationId}`, {
    method: "PUT",
    body: JSON.stringify({ productIds }),
  }).then((r) => r.placements);
}

/** The description text for the clip's pinned products ("" when none). */
export function getProductBlock(generationId: string) {
  return request<{ block: string; count: number }>(
    `/youtube/product-block?generationId=${encodeURIComponent(generationId)}`,
  );
}

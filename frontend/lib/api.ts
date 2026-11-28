// Client-side API wrapper — talks to the Go backend over REST/JSON.
//
// The session lives in an HttpOnly cookie set by the backend, so every request
// goes out with `credentials: "include"`. Set NEXT_PUBLIC_API_BASE_URL to the
// API origin (defaults to the local dev server).

import type {
  Channel,
  ChannelInput,
  Dashboard,
  FalCosts,
  FalCredits,
  Generation,
  Keyframe,
  KeyframeModels,
  Placement,
  Product,
  ProductInput,
  ProductUsage,
  PickableProduct,
  ProductPicture,
  ProductWithUse,
  Profile,
  ProfileInput,
  PromptTemplate,
  PromptTemplateInput,
  CastList,
  VoiceCandidates,
  VoiceSampleResult,
  CastMember,
  CastRole,
  Settings,
  Shot,
  ShotAttempt,
  ShotInput,
  StudioPricing,
  Video,
  YouTubeAccount,
  YouTubePrivacy,
  YouTubeStatus,
  YouTubeUpload,
  RunningItem,
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
  // Something may just have been queued (a clip, a shot, pictures): let the
  // menu bar's "rendering" indicator ask now instead of at its next poll.
  if (
    init?.method === "POST" &&
    /^\/(generations|videos|keyframes)/.test(path) &&
    typeof window !== "undefined"
  ) {
    window.dispatchEvent(new Event("running:refresh"));
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
 *  variables and are pinned to the new clip. `imageUrl` (a pasted link) or
 *  `imageId` (from uploadImage) — not both, "" for none — makes it an
 *  image-to-video clip; only profiles that use fal accept one. */
export function runGeneration(
  promptTemplateId: string,
  opts: { productIds?: string[]; imageUrl?: string; imageId?: string } = {},
) {
  const { productIds = [], imageUrl = "", imageId = "" } = opts;
  return request<{ generation: Generation }>("/generations", {
    method: "POST",
    body: JSON.stringify({ promptTemplateId, productIds, imageUrl, imageId }),
  }).then((r) => r.generation);
}

// --- Images (a starting picture uploaded from this computer) ---

/** Uploads a JPG/PNG; the id goes to runGeneration as `imageId`. Sent as
 *  multipart, so no JSON Content-Type: the browser adds one with the boundary. */
export function uploadImage(file: File) {
  const form = new FormData();
  form.append("file", file);
  return request<{ image: { id: string; contentType: string; sizeBytes: number } }>(
    "/images",
    { method: "POST", body: form, headers: {} },
  ).then((r) => r.image);
}

/** URL to show an uploaded picture in an <img>. The session cookie authorises it. */
export function imageSrc(imageId: string) {
  return `${API_BASE}/images/${imageId}`;
}

/** Mirrors images.MaxBytes on the backend. */
export const MAX_IMAGE_BYTES = 8 * 1024 * 1024;

/** Mirrors maxReferenceBytes on the backend (profiles/cast.go). */
export const MAX_REFERENCE_BYTES = 4 * 1024 * 1024;

// --- Cast (a profile's recurring characters, objects and places) ---

export function listCast(profileId: string) {
  return request<CastList>(`/profiles/${profileId}/cast`);
}

export interface CastInput {
  role: CastRole;
  name: string;
  note: string;
}

export function createCastMember(profileId: string, input: CastInput) {
  return request<{ member: CastMember }>(`/profiles/${profileId}/cast`, {
    method: "POST",
    body: JSON.stringify(input),
  }).then((r) => r.member);
}

export function updateCastMember(profileId: string, castId: string, input: CastInput) {
  return request<{ member: CastMember }>(`/profiles/${profileId}/cast/${castId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.member);
}

export function deleteCastMember(profileId: string, castId: string) {
  return request<void>(`/profiles/${profileId}/cast/${castId}`, { method: "DELETE" });
}

export function addCastImage(profileId: string, castId: string, file: File) {
  const form = new FormData();
  form.append("file", file);
  return request<{ imageId: string }>(
    `/profiles/${profileId}/cast/${castId}/images`,
    { method: "POST", body: form, headers: {} },
  ).then((r) => r.imageId);
}

export function removeCastImage(profileId: string, castId: string, imageId: string) {
  return request<void>(`/profiles/${profileId}/cast/${castId}/images/${imageId}`, {
    method: "DELETE",
  });
}

// --- Keyframes (make the starting picture of a clip with an image model) ---

export function getKeyframeModels() {
  return request<KeyframeModels>("/keyframes/models");
}

/** Asks for `count` pictures. The ticked cast members (`castIds`) go along, with
 *  their pictures named in the prompt, so they stay the same. Returns the
 *  pending request; poll getKeyframe until it has finished. */
export function createKeyframe(input: {
  profileId: string;
  prompt: string;
  model: string;
  count: number;
  /** The profile's cast members that appear in this picture; none = text only. */
  castIds: string[];
  /** Ticked pictures of catalog products (image ids from the product pictures), at
   *  most two different products; named in the prompt with product and label. */
  productPictureIds?: string[];
  /** Stored pictures of the SAME scene to keep (other shots of a video), at most
   *  2, so the new picture shows that scene from another angle or moment. */
  sceneImageIds?: string[];
  /** With scene pictures: keep the camera, room and everything but the pose
   *  fixed (for an END picture, since a clip morphs whatever differs). */
  sameCamera?: boolean;
  /** What the pictures are for ("shot:<id>:start" or ":end"): the request is then
   *  kept after the dialog closes and found again by getLatestKeyframe. */
  context?: string;
}) {
  return request<{ keyframe: Keyframe }>("/keyframes", {
    method: "POST",
    body: JSON.stringify(input),
  }).then((r) => r.keyframe);
}

/** The newest request kept for a place (see `context` above), or null. */
export function getLatestKeyframe(profileId: string, context: string) {
  const q = new URLSearchParams({ profileId, context });
  return request<{ keyframe: Keyframe | null }>(`/keyframes/latest?${q}`).then(
    (r) => r.keyframe,
  );
}

export function getKeyframe(id: string) {
  return request<{ keyframe: Keyframe }>(`/keyframes/${id}`).then(
    (r) => r.keyframe,
  );
}

/** Discards a request and every picture of it that no clip uses. */
export function deleteKeyframe(id: string) {
  return request<void>(`/keyframes/${id}`, { method: "DELETE" });
}

export function deleteGeneration(id: string) {
  return request<void>(`/generations/${id}`, { method: "DELETE" });
}

// --- Videos and shots ---

export function listVideos(profileId: string) {
  return request<{ videos: Video[] }>(
    `/videos?profileId=${encodeURIComponent(profileId)}`,
  ).then((r) => r.videos);
}

export function createVideo(profileId: string, title: string) {
  return request<{ video: Video }>("/videos", {
    method: "POST",
    body: JSON.stringify({ profileId, title }),
  }).then((r) => r.video);
}

/** What a prompt is for: each kind has its own translation rules (a clip prompt
 *  says only what moves, a picture prompt has no style words, an end picture
 *  prompt only what changes). */
export type PromptKind = "motion" | "picture" | "endPicture";

/** Translates one prompt: Thai -> English ("toEnglish", what the models receive)
 *  or English -> Thai ("toThai", so the owner can read what a prompt says). It
 *  returns what the call cost, as fal reported it. */
export function translatePrompt(input: {
  kind: PromptKind;
  direction: "toEnglish" | "toThai";
  text: string;
}) {
  return request<{ text: string; costUsd: number; model: string }>("/videos/translate", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

/** A shot's three prompts rewritten by the script helper (English for the models,
 *  Thai for the owner) and what the call cost. Nothing is saved by the call. */
export interface ShotRevision {
  picturePrompt: string;
  picturePromptTh: string;
  /** "" when the shot has no end picture. */
  endPicturePrompt: string;
  endPicturePromptTh: string;
  /** The clip's motion prompt. */
  prompt: string;
  promptTh: string;
  costUsd: number;
  model: string;
}

/** Asks the script helper to rewrite ONE shot's prompts from a sentence of the
 *  owner's. Saves nothing: the screen puts the result into the shot's boxes. */
export function reviseShot(videoId: string, shotId: string, instruction: string) {
  return request<ShotRevision>(`/videos/${videoId}/shots/${shotId}/revise`, {
    method: "POST",
    body: JSON.stringify({ instruction }),
  });
}

/** Asks the script helper to write a shot list and makes a Video from it: the
 *  scene and each shot's picture prompt, motion prompt and length come filled
 *  in. It makes no pictures and no clips; the only cost is the model's reply,
 *  returned as `costUsd` (fal bills per token, so it is known only afterwards). */
export function draftVideo(input: {
  profileId: string;
  topic: string;
  shots: number;
  seconds: number;
  title?: string;
  /** What kind of video (the structure the AI follows): general (default),
   *  product (needs productId), film or explainer (a narrated educational clip:
   *  `seconds` is then the longest a shot may be). */
  mode?: "general" | "product" | "film" | "explainer";
  /** The catalog product a product video is about; it must have pictures. */
  productId?: string;
}) {
  return request<{ video: Video; costUsd: number; model: string; tooLongShots?: number }>("/videos/draft", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function getVideo(id: string) {
  return request<{ video: Video; shots: Shot[] }>(`/videos/${id}`);
}

export function renameVideo(id: string, title: string) {
  return request<{ video: Video }>(`/videos/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ title }),
  }).then((r) => r.video);
}

/** Deletes the video and its shots; the clips already made stay as ordinary
 *  clips of the profile. */
export function deleteVideo(id: string) {
  return request<void>(`/videos/${id}`, { method: "DELETE" });
}

/** Adds a shot at the end (a new one defaults to 5 seconds). */
export function addShot(videoId: string, input: Partial<ShotInput> = {}) {
  return request<{ shot: Shot }>(`/videos/${videoId}/shots`, {
    method: "POST",
    body: JSON.stringify(input),
  }).then((r) => r.shot);
}

export function updateShot(videoId: string, shotId: string, input: ShotInput) {
  return request<{ shot: Shot }>(`/videos/${videoId}/shots/${shotId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  }).then((r) => r.shot);
}

export function deleteShot(videoId: string, shotId: string) {
  return request<void>(`/videos/${videoId}/shots/${shotId}`, { method: "DELETE" });
}

/** Sets the order; `shotIds` must name every shot of the video once. */
export function reorderShots(videoId: string, shotIds: string[]) {
  return request<void>(`/videos/${videoId}/shots/order`, {
    method: "PUT",
    body: JSON.stringify({ shotIds }),
  });
}

/** Every attempt at a shot, newest first, to compare takes. `chosenId` is the
 *  take the owner picked ("" = the newest successful one is used). */
export function listShotAttempts(videoId: string, shotId: string) {
  return request<{ attempts: ShotAttempt[]; chosenId: string }>(
    `/videos/${videoId}/shots/${shotId}/attempts`,
  );
}

/** Picks which attempt is THE clip of a shot; "" goes back to the newest
 *  successful one. Only a finished attempt of that shot can be chosen. */
export function setShotTake(videoId: string, shotId: string, generationId: string) {
  return request<{ shot: Shot }>(`/videos/${videoId}/shots/${shotId}/take`, {
    method: "PUT",
    body: JSON.stringify({ generationId }),
  }).then((r) => r.shot);
}

/** Marks a shot's plan as looked at and OK (or not). Changing the shot's prompt,
 *  length or picture clears it again. */
export function setShotApproved(videoId: string, shotId: string, approved: boolean) {
  return request<{ shot: Shot }>(`/videos/${videoId}/shots/${shotId}/approved`, {
    method: "PUT",
    body: JSON.stringify({ approved }),
  }).then((r) => r.shot);
}

/** Makes one attempt at each listed shot. Only approved shots are made; the
 *  others come back in `skipped` with the reason. This spends fal credits: ask
 *  first, with the total. */
export function generateShots(videoId: string, shotIds: string[]) {
  return request<{
    queued: { shotId: string; id: string }[];
    skipped: { shotId: string; reason: string }[];
  }>(`/videos/${videoId}/generate`, {
    method: "POST",
    body: JSON.stringify({ shotIds }),
  });
}

/** Makes one attempt at a shot. This spends fal credits: ask first. */
export function generateShot(videoId: string, shotId: string) {
  return request<{ attempt: ShotAttempt }>(
    `/videos/${videoId}/shots/${shotId}/generate`,
    { method: "POST" },
  ).then((r) => r.attempt);
}

// --- Prices and credits ---

export function getStudioPricing() {
  return request<StudioPricing>("/studio/pricing");
}

/** The fal.ai balance. `refresh` asks again right after spending (the backend
 *  still keeps a few seconds' pause between asks of fal). */
export function getFalCredits(refresh = false) {
  return request<FalCredits>(`/credits/fal${refresh ? "?refresh=1" : ""}`);
}

/** Asks fal what these clips really cost and stores the bills it finds. Safe to
 *  call again: clips whose cost is already stored are not asked about. */
export function fetchFalCosts(generationIds: string[]) {
  return request<FalCosts>("/credits/fal/costs", {
    method: "POST",
    body: JSON.stringify({ generationIds }),
  });
}

/** Everything on the home dashboard in one request. Day and month boundaries
 *  follow the browser's time zone. */
export function getDashboard() {
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
  return request<Dashboard>(`/dashboard?tz=${encodeURIComponent(tz)}`);
}

/** What is being made right now (clips and picture requests), for the menu bar. */
export function getRunning() {
  return request<{ items: RunningItem[] }>("/dashboard/running").then((r) => r.items);
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

// --- Product pictures (the real product, for the picture dialog) ---

/** Products that have pictures, with them: what the picture dialog ticks from. */
export function listPickableProducts() {
  return request<{ products: PickableProduct[] }>("/products/pictures").then(
    (r) => r.products,
  );
}

export function listProductPictures(productId: string) {
  return request<{ pictures: ProductPicture[]; max: number }>(
    `/products/${productId}/pictures`,
  );
}

export function addProductPicture(productId: string, file: File, label: string) {
  const form = new FormData();
  form.append("file", file);
  form.append("label", label);
  return request<{ picture: ProductPicture }>(`/products/${productId}/pictures`, {
    method: "POST",
    body: form,
    headers: {},
  }).then((r) => r.picture);
}

export function updateProductPictureLabel(productId: string, imageId: string, label: string) {
  return request<{ picture: ProductPicture }>(
    `/products/${productId}/pictures/${imageId}`,
    { method: "PATCH", body: JSON.stringify({ label }) },
  ).then((r) => r.picture);
}

export function removeProductPicture(productId: string, imageId: string) {
  return request<void>(`/products/${productId}/pictures/${imageId}`, {
    method: "DELETE",
  });
}

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

// --- Thai voice samples (listen to candidate voices before choosing one) ---

export function getVoiceCandidates() {
  return request<VoiceCandidates>("/voice/candidates");
}

/** Every chosen voice speaks the same text. Costs fal credits per character, so
 *  the screen asks first. Can take up to a minute or two. */
export function makeVoiceSamples(input: { text: string; candidates: string[]; customVoices?: string[] }) {
  return request<{ results: VoiceSampleResult[]; chars: number; totalCostUsd: number }>(
    "/voice/samples",
    { method: "POST", body: JSON.stringify(input) },
  );
}

// --- The voice-over of a shot (narration spoken by the narrator voice) ---

/** Speaks the shot's SAVED narration with the narrator voice chosen in Settings,
 *  replacing any earlier audio. Costs fal credits per character, so the screen
 *  asks first. Answers with the updated shot. */
export function makeShotVoice(videoId: string, shotId: string) {
  return request<{ shot: Shot }>(`/videos/${videoId}/shots/${shotId}/voice`, {
    method: "POST",
  }).then((r) => r.shot);
}

/** Removes a shot's voice-over (the narration text stays). */
export function deleteShotVoice(videoId: string, shotId: string) {
  return request<void>(`/videos/${videoId}/shots/${shotId}/voice`, { method: "DELETE" });
}

/** Where the browser plays a shot's voice-over. `version` (the audio's creation
 *  time) changes when it is spoken again, so the browser does not play an old one. */
export const shotVoiceSrc = (videoId: string, shotId: string, version: string) =>
  `${API_BASE}/videos/${videoId}/shots/${shotId}/voice/audio?v=${encodeURIComponent(version)}`;

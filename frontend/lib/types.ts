// Domain types for Phase 1. These mirror what the Go backend will eventually
// return over REST/JSON, so keep them in sync with the backend's sqlc models
// once that exists.

/** Generator a profile defaults to. Keep in sync with generate.Providers() on
 *  the backend. */
export type ProviderKey = "text" | "fal";

export const PROVIDER_OPTIONS: { value: ProviderKey; label: string }[] = [
  { value: "text", label: "Text — prompt only, no cost" },
  { value: "fal", label: "fal.ai video — Kling 3.0" },
];

export interface Profile {
  id: string;
  /** Display name of the channel persona. */
  name: string;
  /** Content niche this persona targets, e.g. "retro gaming history". */
  niche: string;
  /** Free-form notes about tone, audience, do/don't. */
  description: string;
  /** Which generator this profile defaults to. */
  provider: ProviderKey;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** Fields the client sends when creating or editing a Profile. */
export type ProfileInput = Pick<
  Profile,
  "name" | "niche" | "description" | "provider"
>;

export interface Channel {
  id: string;
  /** The profile this channel belongs to. */
  profileId: string;
  /** Display name, e.g. "Main YouTube". */
  name: string;
  /** Target platform, e.g. "youtube", "tiktok". Free text — no integration yet. */
  platform: string;
  /** Account handle, e.g. "@retroarcadevault". */
  handle: string;
  description: string;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** Fields the client sends when creating or editing a Channel. */
export type ChannelInput = Pick<
  Channel,
  "name" | "platform" | "handle" | "description"
>;

export interface PromptTemplate {
  id: string;
  /** The profile this template belongs to. */
  profileId: string;
  /** Short label, e.g. "Cabinet deep-dive". */
  name: string;
  /** The template text, stored verbatim (no variable substitution yet). */
  body: string;
  description: string;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** Fields the client sends when creating or editing a PromptTemplate. */
export type PromptTemplateInput = Pick<
  PromptTemplate,
  "name" | "body" | "description"
>;

export interface Generation {
  id: string;
  profileId: string;
  /** "" if the source template was deleted after this run. */
  promptTemplateId: string;
  /** "" if the source template was deleted. */
  templateName: string;
  /** The exact prompt sent to the provider. */
  inputPrompt: string;
  /** Generated text, or — when outputKind is "video" — a URL. */
  output: string;
  outputKind: "text" | "video";
  status: "pending" | "succeeded" | "failed";
  /** Failure message when status is "failed". */
  error: string;
  provider: string;
  model: string;
  createdAt: string; // ISO 8601
}

/** Global provider credentials, edited on the Settings screen. */
export interface Settings {
  /** fal.ai API key, used by FalVideoGenerator. */
  falApiKey: string;
}

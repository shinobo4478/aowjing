// Domain types for Phase 1. These mirror what the Go backend will eventually
// return over REST/JSON, so keep them in sync with the backend's sqlc models
// once that exists.

export interface Profile {
  id: string;
  /** Display name of the channel persona. */
  name: string;
  /** Content niche this persona targets, e.g. "retro gaming history". */
  niche: string;
  /** Free-form notes about tone, audience, do/don't. */
  description: string;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** Fields the client sends when creating or editing a Profile. */
export type ProfileInput = Pick<Profile, "name" | "niche" | "description">;

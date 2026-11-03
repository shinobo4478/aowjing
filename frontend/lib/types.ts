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

/** Short form of a generator for tight places (roster rows, slate fields). */
export const providerShort = (v: string) =>
  v === "fal" ? "Kling 3.0 video" : v === "text" ? "Text only" : v;

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

// --- YouTube (Phase 2 item 3) ---

export type YouTubePrivacy = "private" | "unlisted" | "public";

/** A channel's YouTube connection, as returned by /youtube/accounts + /status. */
export interface YouTubeAccount {
  channelId: string;
  /** The connected YouTube channel's own id and display name. */
  youtubeChannelId: string;
  youtubeTitle: string;
  connectedAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

export interface YouTubeStatus {
  /** The server has an OAuth client configured at all. */
  configured: boolean;
  connected: boolean;
  account?: YouTubeAccount;
}

/** One publish of a video generation to a connected channel. */
export interface YouTubeUpload {
  id: string;
  generationId: string;
  channelId: string;
  youtubeVideoId: string;
  /** https://www.youtube.com/watch?v=... */
  url: string;
  privacy: YouTubePrivacy;
  title: string;
  createdAt: string; // ISO 8601
}

// --- Products (Phase 3: Shopee baskets) ---

/** A Shopee product in the central catalog — shared by every profile. */
export interface Product {
  id: string;
  name: string;
  /** Canonical Shopee product page (query string stripped). */
  shopeeUrl: string;
  /** False for short links: their product id can't be read without a redirect. */
  idsKnown: boolean;
  /** The owner's own tracked affiliate link; "" until they have one. */
  affiliateUrl: string;
  /** Whole baht; 0 means not set. */
  priceThb: number;
  note: string;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** Fields the client sends when creating or editing a Product. */
export type ProductInput = Pick<
  Product,
  "name" | "shopeeUrl" | "affiliateUrl" | "priceThb" | "note"
>;

/** A catalog product pinned to a clip (generation). `position` is 1 or 2. */
export interface Placement {
  generationId: string;
  position: 1 | 2;
  product: Product;
}

/** How many products one clip can carry. Mirrors products.MaxPerClip. */
export const MAX_PRODUCTS_PER_CLIP = 2;

/** A product as listed in the catalog, with how many clips it is pinned to. */
export interface ProductWithUse extends Product {
  usedCount: number;
}

/** One clip a product is pinned to. */
export interface ProductUsage {
  generationId: string;
  profileId: string;
  profileName: string;
  /** "" if the template was deleted. */
  templateName: string;
  /** 1 or 2: which pin the product is on that clip. */
  position: 1 | 2;
  createdAt: string; // ISO 8601
}

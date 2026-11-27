// Domain types for Phase 1. These mirror what the Go backend will eventually
// return over REST/JSON, so keep them in sync with the backend's sqlc models
// once that exists.

/** Generator a profile defaults to. Keep in sync with generate.Providers() on
 *  the backend. */
export type ProviderKey = "text" | "fal";

export const PROVIDER_OPTIONS: { value: ProviderKey; label: string }[] = [
  { value: "text", label: "Text — prompt only, no cost" },
  { value: "fal", label: "fal.ai video (Kling, Veo)" },
];

/** Short form of a generator for tight places (roster rows, slate fields). */
export const providerShort = (v: string) =>
  v === "fal" ? "fal.ai video" : v === "text" ? "Text only" : v;

/** How a profile's video clips look. Keep in sync with the recipes in
 *  backend/internal/studio — a new style is added there and here. Only used by
 *  profiles whose generator is fal. */
export type VideoStyle = "3d";

export const VIDEO_STYLE_OPTIONS: { value: VideoStyle; label: string }[] = [
  { value: "3d", label: "3D (Pixar-like)" },
];

export const DEFAULT_VIDEO_STYLE: VideoStyle = "3d";

/** Which model makes a profile's clips. Keep in sync with the registry in
 *  backend/internal/studio/models.go — a new model is added there and here. */
export type VideoModel = "kling-v3" | "veo-3.1" | "veo-3.1-fast";

export const VIDEO_MODEL_OPTIONS: { value: VideoModel; label: string }[] = [
  { value: "kling-v3", label: "Kling v3 Standard" },
  { value: "veo-3.1", label: "Veo 3.1" },
  { value: "veo-3.1-fast", label: "Veo 3.1 Fast" },
];

export const DEFAULT_VIDEO_MODEL: VideoModel = "kling-v3";

export const videoModelLabel = (v: string) =>
  VIDEO_MODEL_OPTIONS.find((o) => o.value === v)?.label ?? v;

export const videoStyleLabel = (v: string) =>
  VIDEO_STYLE_OPTIONS.find((o) => o.value === v)?.label ?? v;

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
  /** Which video style its clips use (fal profiles). */
  videoStyle: VideoStyle;
  /** Which video model makes its clips (fal profiles). */
  videoModel: VideoModel;
  /** Whether its clips are made with sound (fal profiles). Off by default. */
  videoAudio: boolean;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** Fields the client sends when creating or editing a Profile. */
export type ProfileInput = Pick<
  Profile,
  "name" | "niche" | "description" | "provider" | "videoStyle" | "videoModel" | "videoAudio"
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
  /** Image the clip starts from (image-to-video); "" for a prompt-only run. */
  imageUrl: string;
  /** Picture uploaded from the owner's computer; "" if none. Served by GET /images/{id}. */
  imageId: string;
  /** Picture the clip was asked to end on; "" if none. */
  endImageId: string;
  /** Generated text, or — when outputKind is "video" — a URL. */
  output: string;
  outputKind: "text" | "video";
  status: "pending" | "succeeded" | "failed";
  /** Failure message when status is "failed". */
  error: string;
  provider: string;
  model: string;
  /** Clip length asked for, in seconds. */
  seconds: number;
  /** The video shot this is an attempt at; "" for a clip made from a template. */
  shotId: string;
  /** What fal actually charged, in USD. null until fal's billing shows it, and
   *  for text-only runs and clips made before this was tracked. */
  costUsd: number | null;
  createdAt: string; // ISO 8601
}

/** Reply of POST /credits/fal/costs: bills found by that call, by generation id. */
export interface FalCosts {
  /** false when no billing key is set in Settings. */
  configured: boolean;
  costs: Record<string, number>;
  error?: string;
}

// --- Videos (Phase 4 item 3): an ordered list of shots ---

/** The most shots one video may have. Mirrors videos.MaxShots. */
export const MAX_SHOTS = 12;

/** A video as listed, with its totals. */
export interface Video {
  id: string;
  profileId: string;
  title: string;
  shotCount: number;
  totalSeconds: number;
  createdAt: string; // ISO 8601
  updatedAt: string; // ISO 8601
}

/** The newest attempt at a shot (a generation). */
export interface ShotAttempt {
  id: string;
  status: "pending" | "succeeded" | "failed";
  /** The clip's URL once succeeded. */
  output: string;
  error: string;
  seconds: number;
  createdAt: string; // ISO 8601
}

/** One shot of a video: the editable plan plus its newest attempt. */
export interface Shot {
  id: string;
  videoId: string;
  /** 1-based position in the video. */
  position: number;
  prompt: string;
  /** What the starting picture should show (Make with AI uses it). Not sent to
   *  the clip model, so editing it does not clear the shot's OK. */
  picturePrompt: string;
  /** What changes by the END picture (a pose or position only); "" for none. */
  endPicturePrompt: string;
  seconds: number;
  /** A public link the shot starts from, or "" . */
  imageUrl: string;
  /** A stored picture the shot starts from (GET /images/{id}), or "". */
  imageId: string;
  /** A stored picture the clip should END on (fal's end_image_url), or "". The
   *  clip flows from the starting picture to this one, so the two must differ
   *  only in the subject's pose. */
  endImageId: string;
  /** The owner OKed this plan on the storyboard, so it may be made in bulk.
   *  Changing the plan clears it. */
  approved: boolean;
  /** null if the shot has never been generated. */
  latest: ShotAttempt | null;
  /** How many attempts there have been; each was paid for. */
  attempts: number;
  /** The clip to USE for this shot: the attempt the owner chose, else the newest
   *  successful one; null while none has succeeded. */
  take: ShotAttempt | null;
  /** True when `take` was chosen by the owner (not just the newest). */
  takeChosen: boolean;
}

/** What the screen sends to change a shot: its whole plan at once. At most one
 *  of imageUrl and imageId is set; neither means no picture yet. */
export interface ShotInput {
  prompt: string;
  picturePrompt: string;
  endPicturePrompt: string;
  seconds: number;
  imageUrl: string;
  imageId: string;
  /** A stored picture to end on; only with a starting picture. "" for none. */
  endImageId: string;
}

/** A picture that gives a profile its own look (character, mascot, style). It
 *  goes to the image model with every keyframe request. */
export interface ReferencePicture {
  /** Served by GET /images/{id}. */
  imageId: string;
  position: number;
}

/** An image model the owner can choose for keyframes (from GET /keyframes/models). */
export interface KeyframeModel {
  key: string;
  label: string;
  /** Price of one picture in US dollars, as fal lists it. */
  priceUsd: number;
}

export interface KeyframeModels {
  models: KeyframeModel[];
  /** The most pictures one request may ask for. */
  maxImages: number;
  /** The most reference pictures a profile keeps. */
  maxReferences: number;
  aspect: string;
}

/** One request to make keyframe pictures (the starting image of a clip). */
export interface Keyframe {
  id: string;
  profileId: string;
  /** Exactly what was sent, with the style's wording added. */
  prompt: string;
  model: string;
  count: number;
  /** How many reference pictures went with it (0 = text only). */
  referenceCount: number;
  status: "pending" | "succeeded" | "failed";
  error: string;
  /** The candidate pictures, once it has succeeded. */
  imageIds: string[];
  createdAt: string; // ISO 8601
}

/** Global provider credentials, edited on the Settings screen. */
export interface Settings {
  /** fal.ai API key, used by FalVideoGenerator. */
  falApiKey: string;
  /** Optional second fal key on the BILLING preset, used only to show the
   *  credit balance in the menu bar. */
  falBillingKey: string;
  /** Optional model id the script helper writes drafts with (e.g.
   *  "anthropic/claude-sonnet-5"); empty means the built-in default. */
  scriptModel: string;
}

/** One video model as GET /studio/pricing describes it. */
export interface PricingModel {
  key: string;
  label: string;
  /** The clip lengths it accepts, in seconds, and the one used when none is chosen. */
  seconds: number[];
  defaultSeconds: number;
  /** USD per second of a silent clip / one with sound; null = not known. Multiply
   *  by the clip's length (assumed to scale linearly; only checked at 5 s). */
  usdPerSecond: number | null;
  usdPerSecondAudio: number | null;
  /** A real bill matched the silent price; otherwise it is fal's list price. */
  confirmedSilent: boolean;
  /** A clip can be made from a prompt alone / given an end picture. */
  supportsText: boolean;
  endPicture: boolean;
}

/** What the clips cost, from GET /studio/pricing. */
export interface StudioPricing {
  models: PricingModel[];
}

/** The pricing entry for a video model key, if the list has loaded. */
export const pricingFor = (p: StudioPricing | null, model: string): PricingModel | undefined =>
  p?.models.find((m) => m.key === model);

/** USD per second for a model with or without sound; undefined when not known. */
export function perSecondUsd(m: PricingModel | undefined, audio: boolean): number | undefined {
  const v = audio ? m?.usdPerSecondAudio : m?.usdPerSecond;
  return v == null ? undefined : v;
}

/** A short caveat to put after a price that has not been seen on a bill. */
export function priceCaveat(m: PricingModel | undefined, audio: boolean): string {
  return m && !audio && m.confirmedSilent
    ? ""
    : " This is fal's list price, not yet seen on a bill.";
}

/** The fal.ai credit balance, from GET /credits/fal. `configured` is false until
 *  a billing key is saved; `error` says why the balance could not be read. */
export interface FalCredits {
  configured: boolean;
  balance?: number;
  currency?: string;
  error?: string;
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

// --- Dashboard (GET /dashboard) ---

export interface Dashboard {
  /** The zone the day/month boundaries were computed in. */
  timezone: string;
  spend: {
    /** Billed so far today (fal can trail a clip by minutes, so treat it as "at least"). */
    todayCost: number;
    todayBilled: number;
    todayUnbilled: number;
    monthCost: number;
    /** Clips this month whose bill fal has shown. */
    monthBilled: number;
    /** fal clips this month with no bill yet (still waiting, or made before bills were tracked). */
    monthUnbilled: number;
    totalCost: number;
    totalBilled: number;
    totalUnbilled: number;
    /** null when nothing is billed yet. */
    avgPerClip: number | null;
    /** 30 days ending today, oldest first, empty days included. */
    days: { date: string; cost: number; clips: number }[];
    byProfile: { profileId: string; name: string; cost: number; clips: number }[];
  };
  active: {
    pending: {
      id: string;
      profileId: string;
      profileName: string;
      seconds: number;
      fromShot: boolean;
      createdAt: string;
    }[];
    failed: {
      id: string;
      profileId: string;
      profileName: string;
      error: string;
      costUsd: number | null;
      createdAt: string;
    }[];
  };
  pipeline: {
    videos: number;
    shots: number;
    shotsApproved: number;
    approvedNotMade: number;
    shotsMade: number;
    unpublishedClips: number;
  };
  published: {
    last7Days: number;
    last30Days: number;
    total: number;
    recent: {
      id: string;
      title: string;
      privacy: string;
      url: string;
      channelName: string;
      profileId: string;
      profileName: string;
      createdAt: string;
    }[];
  };
  products: {
    total: number;
    pinned: number;
    unpinned: { id: string; name: string; createdAt: string }[];
  };
}

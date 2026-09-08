// Dev-only convenience: a "Log in as admin" button on the login screen.
//
// Gated by NEXT_PUBLIC_DEV_LOGIN so it never ships in a real build. The
// credentials it submits come from env too — username defaults to "admin"
// (not sensitive); the password has no hard-coded fallback, so nothing
// secret lives in source. Set NEXT_PUBLIC_DEV_PASSWORD in frontend/.env.local
// (git-ignored) to match your local backend's ADMIN_PASSWORD.
//
// NEXT_PUBLIC_* values are inlined at build time — changing them needs a
// dev-server restart.

export const DEV_LOGIN_ENABLED =
  process.env.NEXT_PUBLIC_DEV_LOGIN === "1" ||
  process.env.NEXT_PUBLIC_DEV_LOGIN === "true";

export const DEV_LOGIN_USERNAME =
  process.env.NEXT_PUBLIC_DEV_USERNAME || "admin";

export const DEV_LOGIN_PASSWORD = process.env.NEXT_PUBLIC_DEV_PASSWORD || "";

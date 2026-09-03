import localFont from "next/font/local";

/**
 * Sukhumvit Set — the project's primary UI font, self-hosted.
 *
 * Files live in ./fonts/ (git-tracked). `next/font/local` subsets them,
 * emits a `size-adjust` fallback to avoid layout shift, and exposes the
 * family through the `--font-sukhumvit` CSS variable, which is referenced by
 * `body` in globals.css and by the antd theme token in providers.tsx.
 */
export const sukhumvit = localFont({
  src: [
    { path: "./fonts/SukhumvitSet-Thin.ttf", weight: "100", style: "normal" },
    { path: "./fonts/SukhumvitSet-Light.ttf", weight: "300", style: "normal" },
    { path: "./fonts/SukhumvitSet-Text.ttf", weight: "400", style: "normal" },
    { path: "./fonts/SukhumvitSet-Medium.ttf", weight: "500", style: "normal" },
    { path: "./fonts/SukhumvitSet-SemiBold.ttf", weight: "600", style: "normal" },
    { path: "./fonts/SukhumvitSet-Bold.ttf", weight: "700", style: "normal" },
  ],
  variable: "--font-sukhumvit",
  display: "swap",
  fallback: [
    "-apple-system",
    "BlinkMacSystemFont",
    "Segoe UI",
    "Leelawadee UI",
    "Noto Sans Thai",
    "Roboto",
    "Helvetica Neue",
    "Arial",
    "sans-serif",
  ],
});

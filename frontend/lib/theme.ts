// Shared theme constants.
//
// Sukhumvit Set is the project's primary UI font, self-hosted via
// next/font/local (see app/fonts.ts). It exposes the `--font-sukhumvit` CSS
// variable, which already expands to the font plus its size-adjusted fallback
// plus the platform Thai fallback stack. This string feeds the antd theme
// token so every antd component uses it too.
//
// Keep in sync with the `body` font-family in app/globals.css.
export const FONT_FAMILY = "var(--font-sukhumvit)";

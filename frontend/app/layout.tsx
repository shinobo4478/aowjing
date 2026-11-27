import type { Metadata, Viewport } from "next";
import AuthGate from "./AuthGate";
import { sukhumvit } from "./fonts";
import Providers from "./providers";
import { THEME_INIT_SCRIPT } from "@/lib/themeScript";
import "./globals.css";

export const metadata: Metadata = {
  title: "AI Content Management Platform",
  description: "Create and organise AI-generated video content.",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  // Dark (OLED-first) is the default; light is the other mode (see
  // app/ThemeMode.tsx). `colorScheme` lets the UA draw native controls and
  // scrollbars to match; `themeColor` tints the mobile browser chrome.
  colorScheme: "dark light",
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#F4F2ED" },
    { media: "(prefers-color-scheme: dark)", color: "#000000" },
  ],
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    // suppressHydrationWarning: the script below sets data-theme before React
    // hydrates (see node_modules/next/dist/docs/.../preventing-flash-before-hydration.md).
    <html lang="en" className={sukhumvit.variable} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }} />
      </head>
      <body>
        <Providers>
          <AuthGate>{children}</AuthGate>
        </Providers>
      </body>
    </html>
  );
}

import type { Metadata } from "next";
import AuthGate from "./AuthGate";
import { sukhumvit } from "./fonts";
import Providers from "./providers";
import "./globals.css";

export const metadata: Metadata = {
  title: "AI Content Management Platform",
  description: "Phase 1 — profiles, channels, and prompt templates.",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={sukhumvit.variable}>
      <body>
        <Providers>
          <AuthGate>{children}</AuthGate>
        </Providers>
      </body>
    </html>
  );
}

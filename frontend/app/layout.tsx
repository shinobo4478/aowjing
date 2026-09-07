import type { Metadata, Viewport } from "next";
import AuthGate from "./AuthGate";
import { sukhumvit } from "./fonts";
import Providers from "./providers";
import "./globals.css";

export const metadata: Metadata = {
  title: "AI Content Management Platform",
  description: "Create and organise AI-generated video content.",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  // OLED-first dark UI. `colorScheme` tells the UA to render native controls,
  // form widgets and scrollbars dark; `themeColor` paints the mobile browser
  // chrome true black so it blends into the app.
  colorScheme: "dark",
  themeColor: "#000000",
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

import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emit a self-contained server (.next/standalone) for a lean Docker image.
  // See frontend/Dockerfile.
  output: "standalone",
};

export default nextConfig;

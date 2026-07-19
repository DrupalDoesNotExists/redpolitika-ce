import type { NextConfig } from "next";

const API_ORIGIN = process.env.REDPOLITIKA_API_ORIGIN ?? "http://127.0.0.1:8080";
const isProd = process.env.NODE_ENV === "production";

const nextConfig: NextConfig = {
  // Static export only for production builds (Go serves `out/`).
  // In `next dev` we keep a normal server so API rewrites work.
  ...(isProd ? { output: "export" as const } : {}),
  trailingSlash: true,
  reactCompiler: true,
  async rewrites() {
    if (isProd) return [];
    return [
      { source: "/api/:path*", destination: `${API_ORIGIN}/api/:path*` },
      { source: "/version", destination: `${API_ORIGIN}/version` },
      { source: "/health", destination: `${API_ORIGIN}/health` },
      { source: "/healthz", destination: `${API_ORIGIN}/healthz` },
    ];
  },
};

export default nextConfig;

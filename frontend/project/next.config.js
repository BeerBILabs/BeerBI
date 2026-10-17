/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Self-contained server in .next/standalone, which the release image copies (docker/frontend/Dockerfile).
  output: 'standalone',
}

// Bundle report: `bun run analyze` (ANALYZE=true, webpack build, writes .next/analyze/*.html).
// Loaded lazily so the dev dependency is not needed for normal builds or at runtime.
const withBundleAnalyzer =
  process.env.ANALYZE === 'true'
    ? (await import('@next/bundle-analyzer')).default({ enabled: true })
    : (config) => config

export default withBundleAnalyzer(nextConfig);

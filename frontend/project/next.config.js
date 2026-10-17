/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
}

// Bundle report: `bun run analyze` (ANALYZE=true, webpack build, writes .next/analyze/*.html).
// Loaded lazily so the dev dependency is not needed for normal builds or at runtime.
const withBundleAnalyzer =
  process.env.ANALYZE === 'true'
    ? (await import('@next/bundle-analyzer')).default({ enabled: true })
    : (config) => config

export default withBundleAnalyzer(nextConfig);

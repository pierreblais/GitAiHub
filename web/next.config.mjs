/** @type {import('next').NextConfig} */
const nextConfig = {
  // A self-contained server for the Docker image (12-factor: one build, many deploys).
  output: "standalone",
  poweredByHeader: false,
  // Compression would buffer the server-sent events of the live feed; the
  // reverse proxy (Caddy) compresses instead.
  compress: false,
  reactStrictMode: true,
};

export default nextConfig;

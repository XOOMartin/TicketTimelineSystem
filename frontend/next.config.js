/** @type {import('next').NextConfig} */
const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8000'

const nextConfig = {
  reactStrictMode: true,
  // Proxy /api/* to the Go server so the browser never makes a
  // cross-origin request, and the session cookie stays first-party.
  async rewrites() {
    return [{ source: '/api/:path*', destination: `${BACKEND_URL}/:path*` }]
  },
}

module.exports = nextConfig

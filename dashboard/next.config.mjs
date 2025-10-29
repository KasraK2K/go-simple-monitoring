/** @type {import('next').NextConfig} */
const monitoringProxyTarget = process.env.NEXT_PUBLIC_MONITORING_BASE_URL;

const nextConfig = {
  reactStrictMode: true,
  experimental: {
    esmExternals: 'loose'
  },
  async rewrites() {
    if (!monitoringProxyTarget) {
      return [];
    }

    return [
      {
        source: '/monitoring',
        destination: `${monitoringProxyTarget}/monitoring`
      },
      {
        source: '/api/v1/server-config',
        destination: `${monitoringProxyTarget}/api/v1/server-config`
      }
    ];
  }
};

export default nextConfig;

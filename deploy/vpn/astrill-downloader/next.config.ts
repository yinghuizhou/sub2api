import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Allow importing playwright from server side
  serverExternalPackages: ['playwright', 'playwright-core'],
};

export default nextConfig;

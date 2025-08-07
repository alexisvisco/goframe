import nextra from "nextra";

const withNextra = nextra({
  // Nextra 4 configuration options
});

export default withNextra({
  // Next.js configuration options
  output: "export",
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  eslint: {
    // Disable ESLint during builds
    ignoreDuringBuilds: true,
  },
  typescript: {
    // Disable TypeScript errors during builds
    ignoreBuildErrors: true,
  },
});

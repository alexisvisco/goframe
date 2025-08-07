import nextra from "nextra";

const withNextra = nextra({
  // Nextra 4 configuration options
});

export default withNextra({
  // Next.js configuration options
  output: "export",
  basePath: process.env.PAGES_BASE_PATH || "",
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    ignoreBuildErrors: true,
  },
});

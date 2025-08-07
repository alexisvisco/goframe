import nextra from "nextra";

const withNextra = nextra({
  // Nextra 4 config
});

export default withNextra({
  output: "export",
  basePath:
    process.env.PAGES_BASE_PATH && process.env.PAGES_BASE_PATH !== ""
      ? process.env.PAGES_BASE_PATH
      : undefined,
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

import nextra from "nextra";

const withNextra = nextra({
  // Nextra 4 config
});

export default withNextra({
  output: "export",
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

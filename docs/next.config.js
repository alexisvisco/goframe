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
});

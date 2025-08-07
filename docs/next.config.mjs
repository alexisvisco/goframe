import nextra from "nextra";

// Configuration Nextra
const withNextra = nextra({
  // Ajoutez ici vos options Nextra si nécessaire
  // theme: 'nextra-theme-docs',
  // themeConfig: './theme.config.tsx'
});

// Détection de l'environnement de production
const isProd = process.env.NODE_ENV === "production";

// Configuration Next.js
const nextConfig = {
  output: "export",
  images: {
    unoptimized: true, // obligatoire pour l'export statique
  },
  // Configuration pour GitHub Pages avec le repository 'goframe'
  basePath: isProd ? "/goframe" : "",
  assetPrefix: isProd ? "/goframe" : "",
};

// Export de la configuration finale
export default withNextra(nextConfig);

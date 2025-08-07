# GoFrame Documentation

This directory contains the GoFrame documentation built with [Nextra](https://nextra.site/).

## Development

To run the documentation locally:

```bash
npm install
npm run dev
```

The documentation will be available at `http://localhost:3000`.

## Building

### Local Build

To build the documentation locally:

```bash
# Use the build script
./build.sh

# Or manually
npm run build
npm run postbuild
```

The static files will be generated in the `out/` directory.

### CI/CD Build

The documentation is automatically built and deployed to GitHub Pages when changes are pushed to the `main` branch. The build process includes:

1. **Build**: Next.js builds the static site
2. **Post-build**: Pagefind generates the search index
3. **Deploy**: Static files are deployed to GitHub Pages

## Deployment

### GitHub Pages (Recommended)

The documentation is automatically deployed to GitHub Pages via GitHub Actions. The workflow is triggered on:

- Push to `main` branch with changes in the `docs/` directory
- Manual workflow dispatch

### Docker Deployment

If you need to deploy using Docker:

1. Build the documentation first:
   ```bash
   ./build.sh
   ```

2. Build the Docker image:
   ```bash
   docker build -t goframe-docs .
   ```

3. Run the container:
   ```bash
   docker run -p 80:80 goframe-docs
   ```

The documentation will be available at `http://localhost`.

## Configuration

- **Next.js Config**: `next.config.js` - Configured for static export
- **Nextra Config**: Configuration is embedded in `next.config.js`
- **Search**: Pagefind is used for client-side search functionality

## Directory Structure

```
docs/
├── app/                 # Documentation content (MDX files)
├── out/                 # Built static files (generated)
├── .next/               # Next.js build cache (generated)
├── package.json         # Dependencies and scripts
├── next.config.js       # Next.js and Nextra configuration
├── Dockerfile          # Docker configuration for nginx
├── build.sh            # Local build script
└── README.md           # This file
```

## Troubleshooting

### Pagefind Issues on ARM

If you encounter issues with Pagefind on ARM architecture, the build is handled by GitHub Actions on Ubuntu (x64) to avoid compatibility issues.

### Search Not Working

Make sure the `postbuild` script runs successfully. The search functionality depends on Pagefind generating the search index in the `out/_pagefind/` directory.

### 404 Errors

Check that:
1. The nginx configuration is correct
2. All static files are properly copied to the web root
3. The `trailingSlash: true` configuration in `next.config.js` matches your server setup
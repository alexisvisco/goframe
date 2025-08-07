#!/bin/bash

# Build script for GoFrame documentation
# This script builds the documentation locally and prepares it for deployment

set -e

echo "🚀 Building GoFrame Documentation..."

# Check if we're in the docs directory
if [ ! -f "package.json" ]; then
    echo "❌ Error: This script must be run from the docs directory"
    exit 1
fi

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf out/
rm -rf .next/

# Install dependencies
echo "📦 Installing dependencies..."
npm ci

# Build the documentation
echo "🔨 Building documentation..."
npm run build

# Run postbuild (pagefind indexing)
echo "🔍 Generating search index with pagefind..."
npm run postbuild

echo "✅ Build completed successfully!"
echo "📁 Static files are ready in the 'out' directory"
echo ""
echo "To test locally, you can serve the 'out' directory with:"
echo "  npx serve out"
echo "  # or"
echo "  python3 -m http.server 3000 --directory out"

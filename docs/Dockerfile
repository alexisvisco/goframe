# Dependencies stage
FROM node:20 AS deps
WORKDIR /app
COPY package.json ./
RUN rm -f package-lock.json
RUN npm install

# Build stage
FROM node:20 AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/package-lock.json ./package-lock.json
COPY . .

# Set environment to avoid telemetry and build issues
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production

# Build the static site and run postbuild (includes pagefind generation)
RUN npm run build

# Production stage - serve static files
FROM nginx:alpine
WORKDIR /usr/share/nginx/html

# Copy the built static files
COPY --from=builder /app/out ./

# Create a custom nginx config for static site
RUN echo 'server { \
    listen 3000; \
    server_name localhost; \
    root /usr/share/nginx/html; \
    index index.html; \
    \
    # Handle pagefind search assets \
    location /_pagefind/ { \
    try_files $uri =404; \
    expires 1h; \
    add_header Cache-Control "public"; \
    } \
    \
    # Handle static assets \
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ { \
    expires 1y; \
    add_header Cache-Control "public, immutable"; \
    try_files $uri =404; \
    } \
    \
    # Handle all other requests \
    location / { \
    try_files $uri $uri.html $uri/ /404.html; \
    } \
    \
    # Handle 404 errors \
    error_page 404 /404.html; \
    location = /404.html { \
    internal; \
    } \
    }' > /etc/nginx/conf.d/default.conf

# Remove default nginx config if it exists
RUN rm -f /etc/nginx/conf.d/default.conf.dpkg-dist

# Expose port 3000
EXPOSE 3000

# Start nginx
CMD ["nginx", "-g", "daemon off;"]

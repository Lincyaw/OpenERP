# Build stage
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /app

# Copy package files
COPY package.json package-lock.json ./

# Install dependencies
RUN npm ci

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Runtime stage - using Nginx to serve static files
FROM nginx:1.25-alpine

# Install runtime dependencies
RUN apk add --no-cache curl

# Copy custom Nginx configuration
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built files from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Create a script to inject runtime env variables
RUN echo '#!/bin/sh' > /docker-entrypoint.sh && \
    echo 'set -e' >> /docker-entrypoint.sh && \
    echo 'echo "window._env_ = {" > /usr/share/nginx/html/env-config.js' >> /docker-entrypoint.sh && \
    echo 'echo "  VITE_API_BASE_URL: \"$VITE_API_BASE_URL\"," >> /usr/share/nginx/html/env-config.js' >> /docker-entrypoint.sh && \
    echo 'echo "  VITE_APP_NAME: \"$VITE_APP_NAME\"," >> /usr/share/nginx/html/env-config.js' >> /docker-entrypoint.sh && \
    echo 'echo "  VITE_APP_VERSION: \"$VITE_APP_VERSION\"" >> /usr/share/nginx/html/env-config.js' >> /docker-entrypoint.sh && \
    echo 'echo "};" >> /usr/share/nginx/html/env-config.js' >> /docker-entrypoint.sh && \
    echo 'exec nginx -g "daemon off;"' >> /docker-entrypoint.sh && \
    chmod +x /docker-entrypoint.sh

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Run nginx
ENTRYPOINT ["/docker-entrypoint.sh"]

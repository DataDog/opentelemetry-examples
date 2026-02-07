FROM node:20-alpine

WORKDIR /app

# Copy dependency manifests first for better Docker layer caching
COPY package*.json ./

# Install production dependencies only
RUN npm install --omit=dev

# Copy application source
COPY controller.js ./

EXPOSE 5002

# Health check to verify the service is responding
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:5002/health || exit 1

# Run as non-root user for security
USER node

CMD ["node", "controller.js"]

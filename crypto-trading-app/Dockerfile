# ============================================================================
# Frontend Dockerfile - Next.js Application
# ============================================================================

# Stage 1: Dependencies
FROM node:20-alpine AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app

# Copy package files
COPY package.json package-lock.json* pnpm-lock.yaml* ./

# Install dependencies (prioritize npm over pnpm)
RUN \
  if [ -f package-lock.json ]; then \
    npm ci; \
  elif [ -f pnpm-lock.yaml ]; then \
    corepack enable pnpm && pnpm install --no-frozen-lockfile; \
  else \
    npm install; \
  fi

# Stage 2: Builder
FROM node:20-alpine AS builder
WORKDIR /app

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Set build-time environment variables (Next.js needs these at build time)
ARG NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_ORDERS_API_URL
ARG NEXT_PUBLIC_MARKET_API_URL
ARG NEXT_PUBLIC_PORTFOLIO_API_URL
ARG NEXT_PUBLIC_SEARCH_API_URL

ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
ENV NEXT_PUBLIC_ORDERS_API_URL=${NEXT_PUBLIC_ORDERS_API_URL}
ENV NEXT_PUBLIC_MARKET_API_URL=${NEXT_PUBLIC_MARKET_API_URL}
ENV NEXT_PUBLIC_PORTFOLIO_API_URL=${NEXT_PUBLIC_PORTFOLIO_API_URL}
ENV NEXT_PUBLIC_SEARCH_API_URL=${NEXT_PUBLIC_SEARCH_API_URL}

# Build the application
RUN \
  if [ -f pnpm-lock.yaml ]; then \
    corepack enable pnpm && pnpm build; \
  elif [ -f package-lock.json ]; then \
    npm run build; \
  else \
    npm run build; \
  fi

# Stage 3: Runner
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create a non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy necessary files
COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

# Set permissions
RUN chown -R nextjs:nodejs /app

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]


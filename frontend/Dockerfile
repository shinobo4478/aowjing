# Frontend image — Next.js standalone output (needs `output: "standalone"`
# in next.config.ts). Final image is just the traced server + static assets.

FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# NEXT_PUBLIC_* is inlined at build time. The API is served same-origin under
# /api by Caddy, so this stays a relative path and never needs a rebuild when
# the domain changes. Keep DEV_LOGIN off in real builds.
ARG NEXT_PUBLIC_API_BASE_URL=/api
ARG NEXT_PUBLIC_DEV_LOGIN=0
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_DEV_LOGIN=$NEXT_PUBLIC_DEV_LOGIN
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:22-alpine AS run
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV HOSTNAME=0.0.0.0
ENV PORT=3000
RUN addgroup -g 1001 -S nodejs && adduser -S nextjs -u 1001
COPY --from=build /app/public ./public
COPY --from=build --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=build --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]

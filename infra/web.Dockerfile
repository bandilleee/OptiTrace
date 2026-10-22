FROM node:22-bookworm-slim@sha256:de170b39439eb0e5a3444ad7d60b015614a381e10fecae2e30ec9e735122f38d AS build
WORKDIR /src
COPY apps/web/package*.json apps/web/.npmrc ./
ARG NPM_LOGLEVEL=warn
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund --fetch-timeout=60000 --fetch-retries=2 --loglevel=$NPM_LOGLEVEL
COPY apps/web/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build
FROM gcr.io/distroless/nodejs22-debian13:nonroot@sha256:b0b1352373f14fe86376d423e6d283ea6b85b912ff55ca89bed8d2fe11c257b8 AS runtime
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3000
WORKDIR /app
COPY --from=build --chown=65532:65532 /src/.next/standalone ./
COPY --from=build --chown=65532:65532 /src/.next/static ./.next/static
COPY --from=build --chown=65532:65532 /src/public ./public
USER 65532:65532
EXPOSE 3000
CMD ["server.js"]

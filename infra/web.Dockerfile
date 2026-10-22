FROM node:22-bookworm-slim@sha256:de170b39439eb0e5a3444ad7d60b015614a381e10fecae2e30ec9e735122f38d AS build
WORKDIR /src
COPY apps/web/package*.json apps/web/.npmrc ./
RUN npm ci
COPY apps/web/ ./
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build
FROM gcr.io/distroless/nodejs22-debian13:nonroot@sha256:b0b1352373f14fe86376d423e6d283ea6b85b912ff55ca89bed8d2fe11c257b8 AS runtime
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3000
WORKDIR /app
COPY --from=build --chown=65532:65532 /src/.next/standalone ./
COPY --from=build --chown=65532:65532 /src/.next/static ./.next/static
USER 65532:65532
EXPOSE 3000
CMD ["server.js"]

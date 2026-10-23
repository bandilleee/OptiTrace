# Local test toolchain only; never used as the production runtime image.
# Prepare the parent with the pinned infra/engine.Dockerfile build target.
ARG ENGINE_BUILDER=optitrace-engine-builder:local
FROM ${ENGINE_BUILDER}
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
ENTRYPOINT ["go", "test", "-race"]

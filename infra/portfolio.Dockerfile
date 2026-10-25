# Portfolio host: a single free web service; no hosted migration or seeding on startup.
FROM golang:1.26.10-alpine@sha256:5cb67f19c5e7994dcb80c074f7cfca8aeeada45bbdca035f73df269cefdcce61 AS build
WORKDIR /src
COPY services/engine/go.mod services/engine/go.sum ./
RUN go mod download
COPY services/engine/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/portfolio ./cmd/portfolio && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/admin ./cmd/admin && \
    CGO_ENABLED=0 go build -trimpath -o /out/healthcheck ./cmd/healthcheck
FROM scratch AS runtime
COPY --from=build /out/ /app/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY infra/certs/supabase-root-2021.crt /app/certs/supabase-root-2021.crt
USER 10001:10001
WORKDIR /app
EXPOSE 10000
# No inherited API ENTRYPOINT: Render's dockerCommand replaces this exec-form CMD.
CMD ["/app/portfolio"]

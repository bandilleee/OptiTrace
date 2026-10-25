FROM golang:1.26.10-alpine@sha256:5cb67f19c5e7994dcb80c074f7cfca8aeeada45bbdca035f73df269cefdcce61 AS build
WORKDIR /src
COPY services/engine/go.mod services/engine/go.sum ./
RUN go mod download
COPY services/engine/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/portfolio ./cmd/portfolio && \
    CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/simulator ./cmd/simulator && \
    CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/admin ./cmd/admin && \
    CGO_ENABLED=0 go build -trimpath -o /out/healthcheck ./cmd/healthcheck
FROM scratch AS runtime
COPY --from=build /out/ /app/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 10001:10001
WORKDIR /app
EXPOSE 8080 8090
ENTRYPOINT ["/app/api"]

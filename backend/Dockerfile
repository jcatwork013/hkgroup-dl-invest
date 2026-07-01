# Build context = repo root (so db/migrations can be copied into the image).
FROM golang:1.22-alpine AS build
ENV GOTOOLCHAIN=local
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/api      ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/migrate  ./cmd/migrate

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
COPY --from=build /out/api     /usr/local/bin/api
COPY --from=build /out/migrate /usr/local/bin/migrate
COPY db/migrations /db/migrations
ENV MIGRATIONS_DIR=/db/migrations
# Encrypted KYC upload storage (volume-mounted at runtime); owned by app so it can write.
RUN mkdir -p /data/uploads && chown -R app /data
ENV UPLOAD_DIR=/data/uploads
USER app
EXPOSE 8080
CMD ["api"]

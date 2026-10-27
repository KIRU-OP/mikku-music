FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mikku ./cmd/mikku

FROM debian:bookworm-slim
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl ffmpeg && rm -rf /var/lib/apt/lists/* \
    && case "$TARGETARCH" in \
       amd64) asset="yt-dlp_linux" ;; \
       arm64) asset="yt-dlp_linux_aarch64" ;; \
       arm) asset="yt-dlp_linux_armv7l" ;; \
       *) echo "Unsupported architecture for yt-dlp: $TARGETARCH" >&2; exit 1 ;; \
       esac \
    && curl -fsSL "https://github.com/yt-dlp/yt-dlp/releases/latest/download/$asset" -o /usr/local/bin/yt-dlp \
    && chmod 0755 /usr/local/bin/yt-dlp \
    && groupadd --system app && useradd --system --gid app --create-home app
WORKDIR /app
COPY --from=build /out/mikku /app/mikku
RUN mkdir -p /app/downloads && chown -R app:app /app
ENV YTDLP_PATH=/usr/local/bin/yt-dlp
ENV DOWNLOAD_DIR=/app/downloads
USER app
CMD ["/app/mikku"]

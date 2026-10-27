FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mikku ./cmd/mikku

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl ffmpeg && rm -rf /var/lib/apt/lists/* \
    && curl -fsSL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux -o /usr/local/bin/yt-dlp \
    && chmod 0755 /usr/local/bin/yt-dlp \
    && groupadd --system app && useradd --system --gid app --create-home app
WORKDIR /app
COPY --from=build /out/mikku /app/mikku
RUN mkdir -p /app/downloads && chown -R app:app /app
ENV YTDLP_PATH=/usr/local/bin/yt-dlp
ENV DOWNLOAD_DIR=/app/downloads
USER app
CMD ["/app/mikku"]

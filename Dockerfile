FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/cantina_band .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=builder /out/cantina_band /app/cantina_band

# Directory where DCA music files should be bind-mounted at runtime,
# e.g. docker run -v /host/path/to/dca:/music ...
ENV MUSIC_DIR=/music
VOLUME ["/music"]

# TOKEN must be provided at runtime, e.g. -e TOKEN=<discord_bot_token>
ENTRYPOINT ["/app/cantina_band"]

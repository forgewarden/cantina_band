FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine3.23@sha256:d9e2f2f07b10cc922da3e80e035c3058810b328d5aef82d2c63680967c5e2ec9 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY main.go ./
COPY discord ./discord
COPY downloader ./downloader

ARG TARGETOS
ARG TARGETARCH
RUN --network=none CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=readonly -trimpath -buildvcs=false -ldflags="-s -w" -o /out/cantina_band .

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
WORKDIR /app

ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED=1970-01-01T00:00:00Z
LABEL org.opencontainers.image.title="Cantina Band" \
      org.opencontainers.image.description="Self-hosted Discord music bot for pre-encoded DCA audio" \
      org.opencontainers.image.source="https://github.com/forgewarden/cantina_band" \
      org.opencontainers.image.url="https://github.com/forgewarden/cantina_band" \
      org.opencontainers.image.documentation="https://github.com/forgewarden/cantina_band#readme" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.revision=$REVISION \
      org.opencontainers.image.created=$CREATED

COPY --from=builder --chown=nonroot:nonroot --chmod=0555 /out/cantina_band /app/cantina_band

ENV MUSIC_DIR=/music

USER nonroot:nonroot
ENTRYPOINT ["/app/cantina_band"]

# Cantina Band

Cantina Band is a self-hosted Discord music bot that streams pre-encoded DCA audio from a local library. It supports fuzzy song matching, a per-server queue, and an optional downloader service for tracks that are not already available locally.

## Requirements

- A Discord bot token
- A directory containing DCA0 files
- Go 1.26.7 or later, or Docker

In the [Discord Developer Portal](https://discord.com/developers/applications), enable **Message Content Intent** under **Bot > Privileged Gateway Intents**. When adding the bot to a server, grant it **View Channels**, **Send Messages**, **Connect**, and **Speak** permissions.

## Run Locally

```bash
git clone https://github.com/forgewarden/cantina-band.git
cd cantina-band
go build -o cantina-band .
./cantina-band -token "<discord-bot-token>" -music-dir "/path/to/music"
```

The same settings can be supplied with environment variables:

```bash
TOKEN="<discord-bot-token>" MUSIC_DIR="/path/to/music" ./cantina-band
```

## Run With Docker

Stable multi-architecture images for `linux/amd64` and `linux/arm64` are published to GitHub Container Registry. Use a version tag for routine deployments:

```bash
docker pull ghcr.io/forgewarden/cantina-band:0.1.0

docker run --detach \
  --name cantina-band \
  --restart unless-stopped \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --cpus 1 \
  --memory 256m \
  --pids-limit 100 \
  --mount type=bind,src="/path/to/discord-token",dst=/run/secrets/discord_token,readonly \
  --mount type=bind,src="/path/to/music",dst=/music,readonly \
  --env TOKEN_FILE=/run/secrets/discord_token \
  ghcr.io/forgewarden/cantina-band:0.1.0
```

The image runs as non-root UID/GID `65532`. The token file and music directory must be readable by that identity. Protect the token file from other host users and never include it in an image or source repository. No inbound port is required.

To build the image locally instead:

```bash
docker build --tag cantina-band:local .
```

An equivalent hardened deployment is available in [`compose.example.yaml`](compose.example.yaml):

```bash
MUSIC_DIR="/path/to/music" \
DISCORD_TOKEN_FILE="/path/to/discord-token" \
docker compose --file compose.example.yaml up --detach
```

## Configuration

Command-line flags take precedence over their corresponding environment variables.

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `-token` | `TOKEN` | Required | Discord bot token |
| None | `TOKEN_FILE` | Disabled | File containing the Discord bot token; used only when `TOKEN` is unset |
| `-music-dir` | `MUSIC_DIR` | Required | Directory containing `.dca` files |
| `-downloader-url` | `DOWNLOADER_URL` | Disabled | Base URL of an optional missing-track downloader |
| `-downloader-timeout` | `DOWNLOADER_TIMEOUT` | `20m` | Timeout for downloader requests |

The downloader is only contacted when no local track matches. It must publish the returned `.dca` file into the same music directory visible to Cantina Band; use a shared volume when the services run in separate containers.

The `-token` flag takes precedence over `TOKEN`, which takes precedence over `TOKEN_FILE`.

## Image Tags and Verification

Stable releases publish the following tags from an annotated `vMAJOR.MINOR.PATCH` Git tag:

| Tag | Behavior |
| --- | --- |
| `0.1.0` | Immutable release tag |
| `0.1` | Newest patch in the minor release |
| `0` | Newest release in the major release |
| `latest` | Newest stable release |
| `sha-<12-character-commit>` | Image built from a specific commit |

For reproducible deployments, resolve and record the release digest rather than relying on a mutable alias:

```bash
IMAGE="ghcr.io/forgewarden/cantina-band:0.1.0"
DIGEST="$(docker buildx imagetools inspect "$IMAGE" --format '{{.Manifest.Digest}}')"
docker pull "ghcr.io/forgewarden/cantina-band@${DIGEST}"
```

Release manifests are signed keylessly by `.github/workflows/release.yml`. Verify the signature with [Cosign](https://docs.sigstore.dev/cosign/system_config/installation/):

```bash
cosign verify \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --certificate-identity-regexp "^https://github.com/forgewarden/cantina-band/.github/workflows/release.yml@refs/tags/v[0-9]+\\.[0-9]+\\.[0-9]+$" \
  "ghcr.io/forgewarden/cantina-band@${DIGEST}"
```

Verify GitHub build provenance with the [GitHub CLI](https://cli.github.com/):

```bash
gh attestation verify \
  "oci://ghcr.io/forgewarden/cantina-band@${DIGEST}" \
  --repo forgewarden/cantina-band
```

Each platform image also includes an attached SBOM and maximum-mode BuildKit provenance.

## Updates and Rollback

Review the GitHub release notes, pull the new immutable version, verify it, and replace the running container. Keep the previously deployed digest in deployment records. Roll back by recreating the container with that previous digest; release tags are never replaced, while `latest`, major, and minor aliases advance with stable releases.

## Publishing Releases

Pull requests and `main` are validated by [`.github/workflows/build.yml`](.github/workflows/build.yml). It runs tests, race detection, static analysis, dependency and vulnerability checks, CodeQL, Dockerfile linting, and non-publishing builds for both supported architectures.

The [release workflow](.github/workflows/release.yml) accepts only annotated stable SemVer tags whose commits are reachable from `main`. It builds each platform once, pushes by digest, scans both digests, creates the multi-platform aliases, attaches provenance, signs the manifest, and creates a GitHub release.

Configure the repository before the first release:

1. Protect `main` and require the `Build` workflow and review before merge.
2. Add a ruleset preventing `v*` tags from being updated or deleted.
3. Create a protected GitHub environment named `release` with a required reviewer.
4. Enable Dependabot alerts, secret scanning, push protection, private vulnerability reporting, and CodeQL.
5. Create and push the first release with `git tag -a v0.1.0 -m "Release v0.1.0"` followed by `git push origin v0.1.0`.
6. After the package is created, link it to this repository and set `ghcr.io/forgewarden/cantina-band` visibility to public.

## Commands

The user running `!play` must be connected to a voice channel.

| Command | Description |
| --- | --- |
| `!play <song>` | Fuzzy-match a local song and play it or add it to the queue |
| `!queue` | Show the current song and up to 10 queued songs |
| `!nowplaying` | Show the current song and queue length |
| `!skip` | Skip the current song |
| `!stop` | Stop playback and clear the queue |

Queues hold up to 10 waiting tracks. The bot disconnects from voice 30 seconds after playback finishes if no new track is added.

## Prepare Music

Only `.dca` files directly inside the music directory are indexed; subdirectories are not scanned. File names, without the `.dca` extension, are used as song names.

Install [FFmpeg](https://ffmpeg.org/) and the [DCA encoder](https://github.com/bwmarrin/dca), then convert a track:

```bash
go install github.com/bwmarrin/dca/cmd/dca@latest
ffmpeg -i input.mp3 -f s16le -ar 48000 -ac 2 pipe:1 | dca > output.dca
```

## License

[MIT](LICENSE)

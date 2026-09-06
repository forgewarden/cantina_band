# Cantina Band

Cantina Band is a self-hosted Discord music bot that streams pre-encoded DCA audio from a local library. It supports fuzzy song matching, a per-server queue, and an optional downloader service for tracks that are not already available locally.

## Requirements

- A Discord bot token
- A directory containing DCA0 files
- Go 1.26 or later, or Docker

In the [Discord Developer Portal](https://discord.com/developers/applications), enable **Message Content Intent** under **Bot > Privileged Gateway Intents**. When adding the bot to a server, grant it **View Channels**, **Send Messages**, **Connect**, and **Speak** permissions.

## Run Locally

```bash
git clone https://github.com/forgewarden/cantina_band.git
cd cantina_band
go build -o cantina_band .
./cantina_band -token "<discord-bot-token>" -music-dir "/path/to/music"
```

The same settings can be supplied with environment variables:

```bash
TOKEN="<discord-bot-token>" MUSIC_DIR="/path/to/music" ./cantina_band
```

## Run With Docker

```bash
docker build -t cantina-band .
docker run --rm \
  -e TOKEN="<discord-bot-token>" \
  -v "/path/to/music:/music" \
  cantina-band
```

## Configuration

Command-line flags take precedence over their corresponding environment variables.

| Flag | Environment variable | Default | Description |
| --- | --- | --- | --- |
| `-token` | `TOKEN` | Required | Discord bot token |
| `-music-dir` | `MUSIC_DIR` | Required | Directory containing `.dca` files |
| `-downloader-url` | `DOWNLOADER_URL` | Disabled | Base URL of an optional missing-track downloader |
| `-downloader-timeout` | `DOWNLOADER_TIMEOUT` | `20m` | Timeout for downloader requests |

The downloader is only contacted when no local track matches. It must publish the returned `.dca` file into the same music directory visible to Cantina Band; use a shared volume when the services run in separate containers.

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

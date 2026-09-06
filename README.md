# Cantina Band

Cantina Band is a Discord bot that can stream music from a local library to your servers voice channels.

## Requirements

- Go 1.26.0 or higher
- Discord Bot Token
- DCA audio files (see conversion section below)

The bot uses message-based commands, so enable **Message Content Intent** for the bot in the Discord Developer Portal under **Bot > Privileged Gateway Intents**.

## Installation

1. Clone the repository:
```bash
git clone https://github.com/forgewarden/cantina_band.git
cd cantina_band
```

2. Build the bot:
```bash
go build -o cantina_band .
```

## Usage

Run the bot with your Discord token and music directory:

```bash
./cantina_band -token <discord_bot_token> -music-dir <path_to_dca_files>
```

Or use environment variables:
```bash
export TOKEN="your_discord_bot_token"
export MUSIC_DIR="/path/to/dca_files"
./cantina_band
```

## Commands

### Music Playback

- **`!play <song>`** - Play a song or add it to the queue
  - Uses fuzzy matching to find the best match
  - Queues song if one is already playing

- **`!stop`** - Stop playback and clear the queue
  - Immediately interrupts current playback
  - Clears all queued songs

- **`!skip`** - Skip to the next song in queue
  - Automatically plays the next queued song

### Queue Management

- **`!queue`** - Display the current queue
  - Shows currently playing song
  - Lists up to 10 queued songs

- **`!nowplaying`** - Show the currently playing song
  - Displays song name and queue count

## Audio File Format

This bot exclusively uses the DCA (Discord Audio) format for optimal performance and compatibility.

### What is DCA?
DCA is a specialized audio format designed for Discord bots:
- Contains Opus-encoded audio frames
- Each frame represents exactly 20ms of audio
- Optimized for streaming to Discord voice channels
- No real-time encoding needed during playback

### Converting Audio to DCA

To convert your audio files (MP3, WAV, etc.) to DCA format, you have several options:

#### Option 1: Using FFmpeg and dca CLI tool

1. Install the dca tool:
```bash
go install github.com/bwmarrin/dca/cmd/dca@latest
```

2. Convert your audio files:
```bash
ffmpeg -i input.mp3 -f s16le -ar 48000 -ac 2 pipe:1 | dca > output.dca
```

#### Option 2: Batch conversion script

Create a script to convert all MP3 files in a directory:
```bash
#!/bin/bash
for file in *.mp3; do
    name="${file%.mp3}"
    ffmpeg -i "$file" -f s16le -ar 48000 -ac 2 pipe:1 | dca > "$name.dca"
done
```

#### Option 3: Using a separate conversion service

Consider running a separate conversion service that watches a directory and automatically converts uploaded audio files to DCA format.

### DCA Conversion Parameters

For optimal quality when creating DCA files:
- Sample Rate: 48000 Hz (required by Discord)
- Channels: 2 (stereo)
- Bitrate: 64-128 kbps (recommended)
- Frame Duration: 20ms

## Troubleshooting

### Bot can't connect to voice channel
- Ensure bot has proper permissions (Connect, Speak, View Channels)
- Check if you're in a voice channel when using commands
- Verify Discord token is valid

### No audio playing
- Ensure DCA files are in the specified music directory
- Check file permissions
- Verify DCA files are properly formatted

### Audio quality issues
- Ensure DCA files were created with correct parameters (48kHz, stereo)
- Check for corrupted DCA files
- Try re-converting the source audio

## Acknowledgments

- Built with [disgo](https://github.com/DisgoOrg/disgo)
- Discord voice E2EE powered by [dave-go](https://github.com/thomas-vilte/dave-go)
- DCA format by [bwmarrin](https://github.com/bwmarrin/dca)
- Fuzzy search powered by [lithammer/fuzzysearch](https://github.com/lithammer/fuzzysearch)

# Cantina Band

Cantina Band is a Discord bot that allows users to stream music from their local library to their current voice channel.

Music is pulled from a provided local directory. Songs should be in `.dca` format.

You can convert `.mp3` and `.wav` files to `.dca` using ffmpeg and [bwmarrin/dca](https://github.com/bwmarrin/dca).

Example:

```
ffmpeg -i test.mp3 -f s16le -ar 48000 -ac 2 pipe:1 | dca > test.dca
```

## Using the bot

```
go run main.go -token <token> -music-dir <dir>
```

## Commands

### `!play <song>`

Plays the song in the current voice channel.

The song that is chosen will be whichever file name best matches the requested song name.

### `!stop`

Stops the current song.

## TODO

- Implement a song queue
- The bot should handle the file conversion
- Add docker file

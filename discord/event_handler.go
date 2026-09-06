package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/disgoorg/disgo/bot"
	discordapi "github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
)

// handleCommands routes all commands to appropriate handlers
func handleCommands(s *bot.Client, m *events.MessageCreate) {
	if m.GuildID == nil {
		return
	}

	// Ignore bot's own messages
	if m.Message.Author.ID == s.ID() {
		return
	}

	// Route to appropriate handler based on prefix
	switch {
	case strings.HasPrefix(m.Message.Content, "!play"):
		songRequestHandler(s, m)
	case strings.HasPrefix(m.Message.Content, "!stop"):
		stopRequestHandler(s, m)
	case strings.HasPrefix(m.Message.Content, "!skip"):
		skipRequestHandler(s, m)
	case strings.HasPrefix(m.Message.Content, "!queue"):
		queueRequestHandler(s, m)
	case strings.HasPrefix(m.Message.Content, "!nowplaying"):
		nowPlayingHandler(s, m)
	}
}

func songRequestHandler(s *bot.Client, m *events.MessageCreate) {
	prefix := "!play"
	songRequest := strings.TrimSpace(m.Message.Content[len(prefix):])

	if songRequest == "" {
		sendMessage(s, m.ChannelID, "No song was requested! Usage: !play <song name>")
		return
	}

	log.Println("user requested song:", songRequest)

	guildID := *m.GuildID
	voiceState, ok := s.Caches.VoiceState(guildID, m.Message.Author.ID)
	if !ok || voiceState.ChannelID == nil {
		sendMessage(s, m.ChannelID, "You need to be in a voice channel to play music!")
		return
	}
	if voiceManager.IsPlaying(guildID) && len(voiceManager.GetQueue(guildID)) >= MaxQueueSize {
		sendMessage(s, m.ChannelID, fmt.Sprintf("Queue is full (max %d songs)", MaxQueueSize))
		return
	}

	// Resolve locally first. The optional downloader only runs for a genuine miss.
	songPath, songName, err := fuzzyFindSong(musicDir, songRequest)
	if errors.Is(err, ErrSongNotFound) && downloaderClient != nil {
		sendMessage(s, m.ChannelID, fmt.Sprintf("**%s** is not in the local library. Downloading it now...", songRequest))

		filename, title, downloadErr := downloaderClient.Download(context.Background(), songRequest)
		if downloadErr == nil {
			songPath, songName, downloadErr = downloadedSong(musicDir, filename, title)
		}
		err = downloadErr
	}
	if err != nil {
		log.Println("error resolving song:", err)
		sendMessage(s, m.ChannelID, fmt.Sprintf("Could not find song: %s", songRequest))
		return
	}

	// Create song request
	song := SongRequest{
		FilePath:         songPath,
		SongName:         songName,
		RequestedBy:      m.Message.Author.ID,
		ChannelID:        *voiceState.ChannelID,
		MessageChannelID: m.ChannelID,
	}

	position, started, err := voiceManager.SubmitSong(s, guildID, song)
	if err != nil {
		sendMessage(s, m.ChannelID, fmt.Sprintf("Error playing song: %v", err))
		return
	}
	if !started {
		sendMessage(s, m.ChannelID,
			fmt.Sprintf("Added **%s** to the queue (Position: %d)", songName, position))
	} else {
		sendMessage(s, m.ChannelID, fmt.Sprintf("Now playing: **%s**", songName))
	}
}

func stopRequestHandler(s *bot.Client, m *events.MessageCreate) {
	guildID := *m.GuildID
	state, exists := voiceManager.GetGuildState(guildID)

	if !exists || state.vc == nil {
		sendMessage(s, m.ChannelID, "Bot is not currently in a voice channel!")
		return
	}

	// Get queue count before clearing
	queueCount := len(voiceManager.GetQueue(guildID))

	if queueCount > 0 {
		sendMessage(s, m.ChannelID,
			fmt.Sprintf("Stopping music and clearing %d queued song(s)...", queueCount))
	} else {
		sendMessage(s, m.ChannelID, "Stopping music...")
	}

	err := voiceManager.StopPlayback(guildID)
	if err != nil {
		log.Println("error stopping playback:", err)
		sendMessage(s, m.ChannelID, fmt.Sprintf("Error stopping playback: %v", err))
	}
}

func skipRequestHandler(s *bot.Client, m *events.MessageCreate) {
	guildID := *m.GuildID
	if !voiceManager.IsPlaying(guildID) {
		sendMessage(s, m.ChannelID, "No song is currently playing!")
		return
	}

	queue := voiceManager.GetQueue(guildID)
	if len(queue) == 0 {
		sendMessage(s, m.ChannelID, "Skipping current song (no more songs in queue)")
	} else {
		sendMessage(s, m.ChannelID,
			fmt.Sprintf("Skipping to next song: **%s**", queue[0].SongName))
	}

	err := voiceManager.SkipSong(guildID)
	if err != nil {
		log.Println("error skipping song:", err)
		sendMessage(s, m.ChannelID, fmt.Sprintf("Error skipping song: %v", err))
	}
}

func queueRequestHandler(s *bot.Client, m *events.MessageCreate) {
	guildID := *m.GuildID
	queue := voiceManager.GetQueue(guildID)

	if len(queue) == 0 {
		// Check if currently playing
		if current, playing := voiceManager.GetCurrentSong(guildID); playing {
			sendMessage(s, m.ChannelID,
				fmt.Sprintf("Currently playing: **%s**\nQueue is empty.", current.SongName))
		} else {
			sendMessage(s, m.ChannelID, "Queue is empty and no song is playing.")
		}
		return
	}

	// Build queue message
	var message strings.Builder

	// Show current song if playing
	if current, playing := voiceManager.GetCurrentSong(guildID); playing {
		message.WriteString(fmt.Sprintf("**Now Playing:** %s\n\n", current.SongName))
	}

	message.WriteString("**Queue:**\n")
	for i, song := range queue {
		message.WriteString(fmt.Sprintf("%d. %s\n", i+1, song.SongName))
		if i >= 9 { // Limit display to 10 items
			break
		}
	}

	sendMessage(s, m.ChannelID, message.String())
}

func nowPlayingHandler(s *bot.Client, m *events.MessageCreate) {
	guildID := *m.GuildID
	current, playing := voiceManager.GetCurrentSong(guildID)

	if !playing {
		sendMessage(s, m.ChannelID, "No song is currently playing.")
		return
	}

	queueCount := len(voiceManager.GetQueue(guildID))
	if queueCount > 0 {
		sendMessage(s, m.ChannelID,
			fmt.Sprintf("**Now Playing:** %s\n(%d song(s) in queue)",
				current.SongName, queueCount))
	} else {
		sendMessage(s, m.ChannelID,
			fmt.Sprintf("**Now Playing:** %s", current.SongName))
	}
}

func sendMessage(client *bot.Client, channelID snowflake.ID, content string) {
	if _, err := client.Rest.CreateMessage(channelID, discordapi.NewMessageCreate().WithContent(content)); err != nil {
		log.Printf("error sending Discord message: %v", err)
	}
}

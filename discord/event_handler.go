package discord

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleCommands routes all commands to appropriate handlers
func handleCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Route to appropriate handler based on prefix
	switch {
	case strings.HasPrefix(m.Content, "!play"):
		songRequestHandler(s, m)
	case strings.HasPrefix(m.Content, "!stop"):
		stopRequestHandler(s, m)
	case strings.HasPrefix(m.Content, "!skip"):
		skipRequestHandler(s, m)
	case strings.HasPrefix(m.Content, "!queue"):
		queueRequestHandler(s, m)
	case strings.HasPrefix(m.Content, "!nowplaying"):
		nowPlayingHandler(s, m)
	}
}

func songRequestHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	prefix := "!play"
	songRequest := strings.TrimSpace(m.Content[len(prefix):])

	if songRequest == "" {
		s.ChannelMessageSend(m.ChannelID, "No song was requested! Usage: !play <song name>")
		return
	}

	log.Println("user requested song:", songRequest)

	// Get user's voice channel
	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		log.Println("error finding guild:", err)
		s.ChannelMessageSend(m.ChannelID, "Error: Could not find server information")
		return
	}

	var channelId string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			if vs.ChannelID == "" {
				s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel to play music!")
				return
			}
			channelId = vs.ChannelID
			break
		}
	}

	if channelId == "" {
		s.ChannelMessageSend(m.ChannelID, "You need to be in a voice channel to play music!")
		return
	}
	if voiceManager.IsPlaying(m.GuildID) && len(voiceManager.GetQueue(m.GuildID)) >= MaxQueueSize {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Queue is full (max %d songs)", MaxQueueSize))
		return
	}

	// Resolve locally first. The optional downloader only runs for a genuine miss.
	songPath, songName, err := fuzzyFindSong(musicDir, songRequest)
	if errors.Is(err, ErrSongNotFound) && downloaderClient != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**%s** is not in the local library. Downloading it now...", songRequest))

		filename, title, downloadErr := downloaderClient.Download(context.Background(), songRequest)
		if downloadErr == nil {
			songPath, songName, downloadErr = downloadedSong(musicDir, filename, title)
		}
		err = downloadErr
	}
	if err != nil {
		log.Println("error resolving song:", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Could not find song: %s", songRequest))
		return
	}

	// Create song request
	song := SongRequest{
		FilePath:         songPath,
		SongName:         songName,
		RequestedBy:      m.Author.ID,
		ChannelID:        channelId,
		MessageChannelID: m.ChannelID,
	}

	position, started, err := voiceManager.SubmitSong(s, m.GuildID, song)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error playing song: %v", err))
		return
	}
	if !started {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Added **%s** to the queue (Position: %d)", songName, position))
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Now playing: **%s**", songName))
	}
}

func stopRequestHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	state, exists := voiceManager.GetGuildState(m.GuildID)

	if !exists || state.vc == nil {
		s.ChannelMessageSend(m.ChannelID, "Bot is not currently in a voice channel!")
		return
	}

	// Get queue count before clearing
	queueCount := len(voiceManager.GetQueue(m.GuildID))

	if queueCount > 0 {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Stopping music and clearing %d queued song(s)...", queueCount))
	} else {
		s.ChannelMessageSend(m.ChannelID, "Stopping music...")
	}

	err := voiceManager.StopPlayback(m.GuildID)
	if err != nil {
		log.Println("error stopping playback:", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error stopping playback: %v", err))
	}
}

func skipRequestHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !voiceManager.IsPlaying(m.GuildID) {
		s.ChannelMessageSend(m.ChannelID, "No song is currently playing!")
		return
	}

	queue := voiceManager.GetQueue(m.GuildID)
	if len(queue) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Skipping current song (no more songs in queue)")
	} else {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Skipping to next song: **%s**", queue[0].SongName))
	}

	err := voiceManager.SkipSong(m.GuildID)
	if err != nil {
		log.Println("error skipping song:", err)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Error skipping song: %v", err))
	}
}

func queueRequestHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	queue := voiceManager.GetQueue(m.GuildID)

	if len(queue) == 0 {
		// Check if currently playing
		if current, playing := voiceManager.GetCurrentSong(m.GuildID); playing {
			s.ChannelMessageSend(m.ChannelID,
				fmt.Sprintf("Currently playing: **%s**\nQueue is empty.", current.SongName))
		} else {
			s.ChannelMessageSend(m.ChannelID, "Queue is empty and no song is playing.")
		}
		return
	}

	// Build queue message
	var message strings.Builder

	// Show current song if playing
	if current, playing := voiceManager.GetCurrentSong(m.GuildID); playing {
		message.WriteString(fmt.Sprintf("**Now Playing:** %s\n\n", current.SongName))
	}

	message.WriteString("**Queue:**\n")
	for i, song := range queue {
		message.WriteString(fmt.Sprintf("%d. %s\n", i+1, song.SongName))
		if i >= 9 { // Limit display to 10 items
			break
		}
	}

	s.ChannelMessageSend(m.ChannelID, message.String())
}

func nowPlayingHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	current, playing := voiceManager.GetCurrentSong(m.GuildID)

	if !playing {
		s.ChannelMessageSend(m.ChannelID, "No song is currently playing.")
		return
	}

	queueCount := len(voiceManager.GetQueue(m.GuildID))
	if queueCount > 0 {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("**Now Playing:** %s\n(%d song(s) in queue)",
				current.SongName, queueCount))
	} else {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("**Now Playing:** %s", current.SongName))
	}
}

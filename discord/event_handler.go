package discord

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func songRequest(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	prefix := "!play"

	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	songRequest := strings.TrimSpace(m.Content[len(prefix):])
	if songRequest == "" {
		log.Println("no song was requested")
		s.ChannelMessageSend(m.ChannelID, "No song was requested!!")
		return
	}

	log.Println("user requested song:", songRequest)

	song, songName, err := fuzzyFindSong(musicDir, songRequest)
	if err != nil {
		log.Println("error finding song:", err)
		return
	}

	err = loadSong(song)
	if err != nil {
		log.Println("error loading song:", err)
		return
	}

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Now playing %s...", songName))

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		log.Println("error finding guild:", err)
		return
	}

	var channelId string

	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			if vs.ChannelID == "" {
				log.Println("user is in a voice state but not a specific channel in guild")
				return
			}
			channelId = vs.ChannelID
		}
	}

	err = playSong(s, m.GuildID, channelId)
	if err != nil {
		log.Println("error playing song:", err)
	}
}

func stopRequest(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	prefix := "!stop"

	if !strings.HasPrefix(m.Content, prefix) {
		return
	}

	s.ChannelMessageSend(m.ChannelID, "Stopping music...")

	guild, err := s.State.Guild(m.GuildID)
	if err != nil {
		log.Println("error finding guild:", err)
		return
	}

	var channelId string

	for _, vs := range guild.VoiceStates {
		if vs.UserID == m.Author.ID {
			if vs.ChannelID == "" {
				log.Println("user is in a voice state but not a specific channel in guild")
				return
			}
			channelId = vs.ChannelID
		}
	}

	err = stopPlaying(s, m.GuildID, channelId)
	if err != nil {
		log.Println("error stopping song:", err)
	}
}

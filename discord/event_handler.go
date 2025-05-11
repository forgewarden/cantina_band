package discord

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if !strings.HasPrefix(m.Content, "!play") {
		return
	}

	err := loadSong(musicDir, "music.dca")
	if err != nil {
		log.Fatal("error loading song,", err)
	}

	s.ChannelMessageSend(m.ChannelID, "Playing music...")

	err = playSong(s, m.GuildID, "470784676831690762")
		if err != nil {
			log.Println("Error playing song:", err)
		}

}

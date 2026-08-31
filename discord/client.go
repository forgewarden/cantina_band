package discord

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var musicDir string
var voiceManager *VoiceManager
var downloaderClient TrackDownloader

type TrackDownloader interface {
	Download(context.Context, string) (string, string, error)
}

func NewBot(token string, dir string, downloader TrackDownloader) (*discordgo.Session, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	musicDir = dir
	voiceManager = NewVoiceManager()
	downloaderClient = downloader

	// Register all command handlers
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Handle all commands in one handler to check prefixes
		handleCommands(s, m)
	})

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	return dg, nil
}

func Run(dg *discordgo.Session) {
	err := dg.Open()
	if err != nil {
		log.Fatal("error opening connection,", err)
	}
	log.Println("connected to websocket")

	defer dg.Close()

	log.Println("bot is now running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Clean up all voice connections on shutdown
	log.Println("Cleaning up voice connections...")
	for guildID := range voiceManager.guilds {
		voiceManager.Disconnect(dg, guildID, "")
	}

	dg.Close()
}

package discord

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var buffer = make([][]byte, 0)
var musicDir string

func NewBot(token string, dir string) (*discordgo.Session, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	musicDir = dir

	dg.AddHandler(songRequest)
	dg.AddHandler(stopRequest)

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

	dg.Close()
}

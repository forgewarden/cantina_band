package discord

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/voice"
	"github.com/thomas-vilte/dave-go/session"
)

var musicDir string
var voiceManager *VoiceManager
var downloaderClient TrackDownloader

type TrackDownloader interface {
	Download(context.Context, string) (string, string, error)
}

func NewBot(token string, dir string, downloader TrackDownloader) (*bot.Client, error) {
	musicDir = dir
	voiceManager = NewVoiceManager()
	downloaderClient = downloader

	return disgo.New(token,
		bot.WithGatewayConfigOpts(gateway.WithIntents(
			gateway.IntentGuilds,
			gateway.IntentGuildMessages,
			gateway.IntentGuildVoiceStates,
			gateway.IntentMessageContent,
		)),
		bot.WithCacheConfigOpts(cache.WithCaches(cache.FlagVoiceStates)),
		bot.WithEventManagerConfigOpts(
			bot.WithAsyncEventsEnabled(),
			bot.WithListenerFunc(func(event *events.MessageCreate) {
				handleCommands(event.Client(), event)
			}),
		),
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(session.CreateFunc()),
		),
	)
}

func Run(client *bot.Client) {
	ctx := context.Background()
	err := client.OpenGateway(ctx)
	if err != nil {
		log.Fatal("error opening connection,", err)
	}
	log.Println("connected to websocket")

	log.Println("bot is now running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
	signal.Stop(sc)

	log.Println("cleaning up Discord connections...")
	client.Close(ctx)
}

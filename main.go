package main

import (
	"log"
	"flag"
	"os"
	"github.com/forgewarden/cantina_band/m/discord"
)

func main() {
	token := flag.String("token", lookupEnvOrString("TOKEN", ""), "Discord bot token")
	musicDir := flag.String("music-dir", lookupEnvOrString("MUSIC_DIR", ""), "Directory containing music files")
	flag.Parse()

	if *token == "" {
		log.Panic("no token provided")
	}

	if *musicDir == "" {
		log.Panic("no music directory provided")
	}


	bot, err := discord.NewBot(*token, *musicDir)
	if err != nil {
		log.Fatal("error creating bot,", err)
	}

	log.Println("bot created")

	discord.Run(bot)
}

func lookupEnvOrString(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}


package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/forgewarden/cantina-band/discord"
	"github.com/forgewarden/cantina-band/downloader"
)

func main() {
	tokenDefault, err := lookupToken()
	if err != nil {
		log.Fatal(err)
	}
	token := flag.String("token", tokenDefault, "Discord bot token")
	musicDir := flag.String("music-dir", lookupEnvOrString("MUSIC_DIR", ""), "Directory containing music files")
	downloaderURL := flag.String("downloader-url", lookupEnvOrString("DOWNLOADER_URL", ""), "Optional URL for the missing-track downloader service")
	downloaderTimeoutValue := flag.String("downloader-timeout", lookupEnvOrString("DOWNLOADER_TIMEOUT", "20m"), "Timeout for remote track resolution and download")
	flag.Parse()

	if *token == "" {
		log.Panic("no token provided")
	}

	if *musicDir == "" {
		log.Panic("no music directory provided")
	}
	downloaderTimeout, err := time.ParseDuration(*downloaderTimeoutValue)
	if err != nil || downloaderTimeout <= 0 {
		log.Panic("invalid downloader timeout")
	}
	downloaderClient, err := downloader.NewClient(*downloaderURL, downloaderTimeout)
	if err != nil {
		log.Fatal("error creating downloader client,", err)
	}

	bot, err := discord.NewBot(*token, *musicDir, downloaderClient)
	if err != nil {
		log.Fatal("error creating bot,", err)
	}

	log.Println("bot created")

	discord.Run(bot)
}

func lookupToken() (string, error) {
	if token, ok := os.LookupEnv("TOKEN"); ok {
		return token, nil
	}

	tokenFile, ok := os.LookupEnv("TOKEN_FILE")
	if !ok {
		return "", nil
	}

	token, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", fmt.Errorf("read token file: %w", err)
	}
	return strings.TrimSpace(string(token)), nil
}

func lookupEnvOrString(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

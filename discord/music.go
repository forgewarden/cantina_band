package discord

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

func fuzzyFindSong(musicDir string, songName string) (string, string, error) {
	fileMap := map[string]string{}

	musicDir = musicDir + "/"
	files, err := os.ReadDir(musicDir)
	if err != nil {
		log.Println("Error reading music musicDirectory :", err)
		return "", "", err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		extension := filepath.Ext(file.Name())
		songName := strings.TrimSuffix(file.Name(), extension)
		fileMap[file.Name()] = songName
	}

	fileNames := []string{}
	for name := range fileMap {
		fileNames = append(fileNames, name)
	}

	matches := fuzzy.Ranks{}
	matches = fuzzy.RankFindNormalizedFold(songName, fileNames)
	sort.Sort(matches)

	song := (musicDir + matches[0].Target)

	return song, fileMap[matches[0].Target], nil
}

func loadSong(song string) error {

	file, err := os.Open(song)
	if err != nil {
		log.Println("Error opening dca file :", err)
		return err
	}

	var opuslen int16

	for {
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err := file.Close()
			if err != nil {
				return err
			}
			return nil
		}

		if err != nil {
			log.Println("Error reading from dca file :", err)
			return err
		}

		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		if err != nil {
			log.Println("Error reading from dca file :", err)
			return err
		}

		buffer = append(buffer, InBuf)
	}
}

func playSong(s *discordgo.Session, guildID, channelID string) (err error) {

	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	time.Sleep(250 * time.Millisecond)

	vc.Speaking(true)

	for _, buff := range buffer {
		vc.OpusSend <- buff
	}

	vc.Speaking(false)

	time.Sleep(250 * time.Millisecond)

	vc.Disconnect()

	return nil
}

func stopPlaying(s *discordgo.Session, guildID, channelID string) (err error) {

	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	time.Sleep(250 * time.Millisecond)

	vc.Speaking(false)

	time.Sleep(250 * time.Millisecond)

	vc.Disconnect()

	return nil
}

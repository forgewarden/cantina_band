package discord

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

func loadSong(musicDir string, songName string) error {

	musicDir = musicDir + "/"
	files, err := os.ReadDir(musicDir)
	if err != nil {
		log.Println("Error reading music musicDirectory :", err)
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if file.Name() == songName {
			file, err := os.Open(musicDir + file.Name())
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
	}
	return nil
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

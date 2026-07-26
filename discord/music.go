package discord

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

func fuzzyFindSong(musicDir string, songName string) (string, string, error) {
	fileMap := map[string]string{}

	musicDir = musicDir + "/"

	files, err := os.ReadDir(musicDir)
	if err != nil {
		return "", "", err
	}

	// Look for .dca files only
	for _, file := range files {
		fileName := file.Name()
		if !file.IsDir() && strings.HasSuffix(fileName, ".dca") {
			fileKey := strings.TrimSuffix(fileName, ".dca")
			fileMap[fileKey] = fileName
		}
	}

	if len(fileMap) == 0 {
		return "", "", fmt.Errorf("no .dca files found in music directory")
	}

	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}

	matches := fuzzy.RankFindFold(songName, keys)
	sort.Sort(matches)

	if len(matches) == 0 {
		return "", "", fmt.Errorf("no matching song found")
	}

	bestMatch := matches[0].Target
	filePath := musicDir + fileMap[bestMatch]

	return filePath, bestMatch, nil
}

func loadSong(song string) error {
	file, err := os.Open(song)
	if err != nil {
		log.Println("Error opening dca file :", err)
		return err
	}
	defer file.Close()

	var opuslen int16

	// Clear existing buffer
	buffer = make([][]byte, 0)

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





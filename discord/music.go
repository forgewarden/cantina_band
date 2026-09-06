package discord

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

var ErrSongNotFound = errors.New("song not found")

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
		return "", "", fmt.Errorf("%w: no .dca files in music directory", ErrSongNotFound)
	}

	keys := make([]string, 0, len(fileMap))
	for k := range fileMap {
		keys = append(keys, k)
	}

	matches := fuzzy.RankFindFold(songName, keys)
	sort.Sort(matches)

	if len(matches) == 0 {
		return "", "", ErrSongNotFound
	}

	bestMatch := matches[0].Target
	filePath := musicDir + fileMap[bestMatch]

	return filePath, bestMatch, nil
}

func downloadedSong(libraryDir, filename, title string) (string, string, error) {
	if filename == "" || filepath.Base(filename) != filename || filepath.Ext(filename) != ".dca" {
		return "", "", fmt.Errorf("downloader returned invalid filename %q", filename)
	}

	filePath := filepath.Join(libraryDir, filename)
	info, err := os.Lstat(filePath)
	if err != nil {
		return "", "", fmt.Errorf("downloaded song is unavailable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("downloaded song %q is not a regular file", filename)
	}
	if title == "" {
		title = filename[:len(filename)-len(filepath.Ext(filename))]
	}
	return filePath, title, nil
}

func loadSong(song string) ([][]byte, error) {
	file, err := os.Open(song)
	if err != nil {
		log.Println("Error opening dca file :", err)
		return nil, err
	}
	defer file.Close()

	var opuslen uint16
	buffer := make([][]byte, 0)

	for {
		err = binary.Read(file, binary.LittleEndian, &opuslen)

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return buffer, nil
		}

		if err != nil {
			log.Println("Error reading from dca file :", err)
			return nil, err
		}
		if opuslen == 0 || opuslen > 4096 {
			return nil, fmt.Errorf("invalid Opus frame length %d", opuslen)
		}

		InBuf := make([]byte, opuslen)
		err = binary.Read(file, binary.LittleEndian, &InBuf)

		if err != nil {
			log.Println("Error reading from dca file :", err)
			return nil, err
		}

		buffer = append(buffer, InBuf)
	}
}

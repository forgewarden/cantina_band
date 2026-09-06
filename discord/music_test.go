package discord

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFuzzyFindSongReportsLocalMiss(t *testing.T) {
	library := t.TempDir()

	_, _, err := fuzzyFindSong(library, "missing")
	if !errors.Is(err, ErrSongNotFound) {
		t.Fatalf("expected ErrSongNotFound, got %v", err)
	}
}

func TestFuzzyFindSongReturnsLocalTrack(t *testing.T) {
	library := t.TempDir()
	filePath := filepath.Join(library, "Cantina Band.dca")
	if err := os.WriteFile(filePath, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	gotPath, gotName, err := fuzzyFindSong(library, "cantina")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != filePath || gotName != "Cantina Band" {
		t.Fatalf("unexpected match: %q, %q", gotPath, gotName)
	}
}

func TestDownloadedSongRequiresPublishedLibraryFile(t *testing.T) {
	library := t.TempDir()
	filename := "Cantina Band [abc123].dca"
	if err := os.WriteFile(filepath.Join(library, filename), []byte("dca"), 0o600); err != nil {
		t.Fatal(err)
	}

	filePath, title, err := downloadedSong(library, filename, "Cantina Band")
	if err != nil {
		t.Fatal(err)
	}
	if filePath != filepath.Join(library, filename) || title != "Cantina Band" {
		t.Fatalf("unexpected resolved song: %q, %q", filePath, title)
	}

	if _, _, err := downloadedSong(library, "../escape.dca", "Escape"); err == nil {
		t.Fatal("expected path traversal filename to be rejected")
	}
}

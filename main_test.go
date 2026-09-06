package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupTokenPrefersEnvironment(t *testing.T) {
	t.Setenv("TOKEN", "environment-token")
	t.Setenv("TOKEN_FILE", filepath.Join(t.TempDir(), "missing"))

	token, err := lookupToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "environment-token" {
		t.Fatalf("unexpected token %q", token)
	}
}

func TestLookupTokenReadsFile(t *testing.T) {
	unsetEnv(t, "TOKEN")
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("file-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TOKEN_FILE", tokenFile)

	token, err := lookupToken()
	if err != nil {
		t.Fatal(err)
	}
	if token != "file-token" {
		t.Fatalf("unexpected token %q", token)
	}
}

func TestLookupTokenReportsFileError(t *testing.T) {
	unsetEnv(t, "TOKEN")
	t.Setenv("TOKEN_FILE", filepath.Join(t.TempDir(), "missing"))

	if _, err := lookupToken(); err == nil {
		t.Fatal("expected token file error")
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, value)
		} else {
			os.Unsetenv(key)
		}
	})
}

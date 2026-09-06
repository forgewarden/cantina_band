package discord

import (
	"testing"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

func TestAutoDisconnectUsesConfiguredDelay(t *testing.T) {
	manager := NewVoiceManager(10 * time.Millisecond)
	guildID := snowflake.ID(1)
	manager.GetOrCreateGuildState(guildID)
	manager.StartAutoDisconnectTimer(nil, guildID, 0)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, exists := manager.GetGuildState(guildID); !exists {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("guild was not disconnected after the configured delay")
}

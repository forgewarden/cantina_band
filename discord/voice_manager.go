package discord

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/snowflake/v2"
)

const (
	MaxQueueSize = 10
)

type SongRequest struct {
	FilePath         string
	SongName         string
	RequestedBy      snowflake.ID
	ChannelID        snowflake.ID
	MessageChannelID snowflake.ID // Channel to send messages to
}

type GuildVoiceState struct {
	vc                  voice.Conn
	isPlaying           bool
	stopChan            chan struct{}
	skipChan            chan struct{}
	queue               []SongRequest
	queueMutex          sync.Mutex
	admissionMutex      sync.Mutex
	currentSong         *SongRequest
	lastActivity        time.Time
	autoDisconnectTimer *time.Timer
	timerMutex          sync.Mutex
}

type VoiceManager struct {
	guilds              map[snowflake.ID]*GuildVoiceState
	mu                  sync.RWMutex
	autoDisconnectDelay time.Duration
}

// NewVoiceManager creates a new voice manager instance
func NewVoiceManager(autoDisconnectDelay time.Duration) *VoiceManager {
	return &VoiceManager{
		guilds:              make(map[snowflake.ID]*GuildVoiceState),
		autoDisconnectDelay: autoDisconnectDelay,
	}
}

// GetOrCreateGuildState gets or creates a guild voice state
func (vm *VoiceManager) GetOrCreateGuildState(guildID snowflake.ID) *GuildVoiceState {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if state, exists := vm.guilds[guildID]; exists {
		return state
	}

	state := &GuildVoiceState{
		queue:        make([]SongRequest, 0),
		lastActivity: time.Now(),
	}
	vm.guilds[guildID] = state
	return state
}

// GetGuildState gets guild state without creating
func (vm *VoiceManager) GetGuildState(guildID snowflake.ID) (*GuildVoiceState, bool) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	state, exists := vm.guilds[guildID]
	return state, exists
}

// JoinChannel joins a voice channel
func (vm *VoiceManager) JoinChannel(client *bot.Client, guildID, channelID snowflake.ID) error {
	state := vm.GetOrCreateGuildState(guildID)

	// If already connected to the same channel, reuse connection
	if state.vc != nil {
		return nil
	}

	ctx := context.Background()
	vc := client.VoiceManager.CreateConn(guildID)
	err := vc.Open(ctx, channelID, false, true)
	if err != nil {
		return fmt.Errorf("failed to join voice channel: %w", err)
	}

	state.vc = vc
	state.lastActivity = time.Now()
	vm.CancelAutoDisconnectTimer(guildID)

	return nil
}

// QueueSong adds a song to the queue
func (vm *VoiceManager) QueueSong(guildID snowflake.ID, song SongRequest) (int, error) {
	state := vm.GetOrCreateGuildState(guildID)

	state.queueMutex.Lock()
	defer state.queueMutex.Unlock()

	if len(state.queue) >= MaxQueueSize {
		return -1, fmt.Errorf("queue is full (max %d songs)", MaxQueueSize)
	}

	state.queue = append(state.queue, song)
	return len(state.queue), nil
}

// GetNextSong gets and removes the next song from queue
func (vm *VoiceManager) GetNextSong(guildID snowflake.ID) (SongRequest, bool) {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return SongRequest{}, false
	}

	state.queueMutex.Lock()
	defer state.queueMutex.Unlock()

	if len(state.queue) == 0 {
		return SongRequest{}, false
	}

	song := state.queue[0]
	state.queue = state.queue[1:]
	return song, true
}

// GetQueue returns a copy of the current queue
func (vm *VoiceManager) GetQueue(guildID snowflake.ID) []SongRequest {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return []SongRequest{}
	}

	state.queueMutex.Lock()
	defer state.queueMutex.Unlock()

	queue := make([]SongRequest, len(state.queue))
	copy(queue, state.queue)
	return queue
}

// ClearQueue clears the queue for a guild
func (vm *VoiceManager) ClearQueue(guildID snowflake.ID) int {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return 0
	}

	state.queueMutex.Lock()
	defer state.queueMutex.Unlock()

	count := len(state.queue)
	state.queue = []SongRequest{}
	return count
}

// SubmitSong atomically decides whether to start a song or append it to the guild queue.
func (vm *VoiceManager) SubmitSong(client *bot.Client, guildID snowflake.ID, song SongRequest) (int, bool, error) {
	state := vm.GetOrCreateGuildState(guildID)
	state.admissionMutex.Lock()
	defer state.admissionMutex.Unlock()

	if state.isPlaying {
		position, err := vm.QueueSong(guildID, song)
		return position, false, err
	}
	if err := vm.PlaySong(client, guildID, song); err != nil {
		return 0, false, err
	}
	return 0, true, nil
}

// PlaySong starts playing a song
func (vm *VoiceManager) PlaySong(client *bot.Client, guildID snowflake.ID, song SongRequest) error {
	state := vm.GetOrCreateGuildState(guildID)

	// Cancel any disconnect timer
	vm.CancelAutoDisconnectTimer(guildID)

	// Join channel if not connected
	if state.vc == nil {
		err := vm.JoinChannel(client, guildID, song.ChannelID)
		if err != nil {
			return err
		}
	}

	// Set up channels for control
	state.stopChan = make(chan struct{})
	state.skipChan = make(chan struct{})
	state.isPlaying = true
	state.currentSong = &song
	state.lastActivity = time.Now()

	// Start playback in goroutine
	go vm.playbackWorker(client, guildID, state, song)

	return nil
}

// playbackWorker handles the actual audio streaming
func (vm *VoiceManager) playbackWorker(client *bot.Client, guildID snowflake.ID, state *GuildVoiceState, song SongRequest) {
	defer func() {
		state.admissionMutex.Lock()
		defer state.admissionMutex.Unlock()

		state.isPlaying = false
		state.currentSong = nil

		// Process queue or start disconnect timer
		if nextSong, hasNext := vm.GetNextSong(guildID); hasNext {
			// Send message about next song
			if song.MessageChannelID != 0 {
				sendMessage(client, song.MessageChannelID,
					fmt.Sprintf("Now playing: **%s**", nextSong.SongName))
			}

			// Play next song
			err := vm.PlaySong(client, guildID, nextSong)
			if err != nil {
				log.Printf("Error playing next song: %v", err)
				if song.MessageChannelID != 0 {
					sendMessage(client, song.MessageChannelID,
						fmt.Sprintf("Error playing next song: %v", err))
				}
			}
		} else {
			// No more songs, start disconnect timer
			vm.StartAutoDisconnectTimer(client, guildID, song.MessageChannelID)
		}
	}()

	// Load the song
	audioBuffer, err := loadSong(song.FilePath)
	if err != nil {
		log.Printf("Error loading song %s: %v", song.SongName, err)
		if song.MessageChannelID != 0 {
			sendMessage(client, song.MessageChannelID,
				fmt.Sprintf("Failed to load **%s**: %v", song.SongName, err))
		}
		return
	}

	// Small delay before starting
	time.Sleep(250 * time.Millisecond)

	// Start speaking
	if err := state.vc.SetSpeaking(context.Background(), voice.SpeakingFlagMicrophone); err != nil {
		log.Printf("Error setting speaking state: %v", err)
	}
	defer func() {
		if err := state.vc.SetSpeaking(context.Background(), voice.SpeakingFlagNone); err != nil {
			log.Printf("Error clearing speaking state: %v", err)
		}
	}()

	// Stream audio with interrupt checking
	interrupted := vm.streamAudioInterruptible(state.vc, audioBuffer, state.stopChan, state.skipChan)

	if interrupted {
		log.Printf("Playback interrupted for guild %s", guildID)
	}

	// Small delay after stopping
	time.Sleep(250 * time.Millisecond)
}

// streamAudioInterruptible streams audio with ability to interrupt
func (vm *VoiceManager) streamAudioInterruptible(vc voice.Conn,
	audioBuffer [][]byte, stopChan, skipChan chan struct{}) bool {

	// Create a ticker for precise 20ms frame timing
	// Each Opus frame represents 20ms of audio at 48kHz
	frameTicker := time.NewTicker(20 * time.Millisecond)
	defer frameTicker.Stop()

	for _, buff := range audioBuffer {
		select {
		case <-stopChan:
			// Stop was requested - clear everything
			return true
		case <-skipChan:
			// Skip was requested - move to next
			return true
		case <-frameTicker.C:
			if _, err := vc.UDP().Write(buff); err != nil {
				log.Printf("Error sending audio frame: %v", err)
				return true
			}
		}
	}

	return false // Completed normally
}

// StopPlayback stops current playback and clears queue
func (vm *VoiceManager) StopPlayback(guildID snowflake.ID) error {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return fmt.Errorf("not connected to any voice channel")
	}

	state.admissionMutex.Lock()
	defer state.admissionMutex.Unlock()

	// Clear the queue
	vm.ClearQueue(guildID)

	// Signal stop if playing
	if state.isPlaying && state.stopChan != nil {
		close(state.stopChan)
		state.isPlaying = false
	}

	// Start disconnect timer
	vm.StartAutoDisconnectTimer(nil, guildID, 0)

	return nil
}

// SkipSong skips the current song
func (vm *VoiceManager) SkipSong(guildID snowflake.ID) error {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return fmt.Errorf("not connected to any voice channel")
	}

	state.admissionMutex.Lock()
	defer state.admissionMutex.Unlock()

	if !state.isPlaying {
		return fmt.Errorf("no song is currently playing")
	}

	// Signal skip if playing
	if state.skipChan != nil {
		close(state.skipChan)
	}

	return nil
}

// StartAutoDisconnectTimer starts the auto-disconnect timer
func (vm *VoiceManager) StartAutoDisconnectTimer(client *bot.Client, guildID snowflake.ID, messageChannelID snowflake.ID) {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return
	}

	state.timerMutex.Lock()
	defer state.timerMutex.Unlock()

	// Cancel existing timer
	if state.autoDisconnectTimer != nil {
		state.autoDisconnectTimer.Stop()
	}

	// Send notification if channel provided
	if messageChannelID != 0 && client != nil {
		sendMessage(client, messageChannelID,
			fmt.Sprintf("No more songs in queue. Will disconnect in %s if no new songs are added.", vm.autoDisconnectDelay))
	}

	// Start new timer
	state.autoDisconnectTimer = time.AfterFunc(vm.autoDisconnectDelay, func() {
		vm.Disconnect(client, guildID, messageChannelID)
	})
}

// CancelAutoDisconnectTimer cancels the auto-disconnect timer
func (vm *VoiceManager) CancelAutoDisconnectTimer(guildID snowflake.ID) {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return
	}

	state.timerMutex.Lock()
	defer state.timerMutex.Unlock()

	if state.autoDisconnectTimer != nil {
		state.autoDisconnectTimer.Stop()
		state.autoDisconnectTimer = nil
	}
}

// Disconnect disconnects from voice and cleans up
func (vm *VoiceManager) Disconnect(client *bot.Client, guildID snowflake.ID, messageChannelID snowflake.ID) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	state, exists := vm.guilds[guildID]
	if !exists {
		return
	}

	// Cancel any timers
	if state.autoDisconnectTimer != nil {
		state.autoDisconnectTimer.Stop()
	}

	// Disconnect from voice
	if state.vc != nil {
		ctx := context.Background()
		state.vc.Close(ctx)
		state.vc = nil
	}

	// Clear queue
	state.queue = []SongRequest{}

	// Remove from map
	delete(vm.guilds, guildID)

	// Send disconnection message if channel provided
	if messageChannelID != 0 && client != nil {
		sendMessage(client, messageChannelID, "Disconnected from voice channel.")
	}
}

// GetCurrentSong returns the currently playing song
func (vm *VoiceManager) GetCurrentSong(guildID snowflake.ID) (*SongRequest, bool) {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return nil, false
	}
	state.admissionMutex.Lock()
	defer state.admissionMutex.Unlock()
	if state.currentSong == nil {
		return nil, false
	}
	return state.currentSong, true
}

// IsPlaying checks if a guild is currently playing
func (vm *VoiceManager) IsPlaying(guildID snowflake.ID) bool {
	state, exists := vm.GetGuildState(guildID)
	if !exists {
		return false
	}
	state.admissionMutex.Lock()
	defer state.admissionMutex.Unlock()
	return state.isPlaying
}

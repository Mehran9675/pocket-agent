// Package local runs language models on the device: GGUF models
// executed by llama.cpp (linked in with the "llama" build tag, see
// scripts/build-llama.sh), a catalog of models sized for phones and
// tablets, and a resumable downloader for Hugging Face.
package local

import (
	"context"
	"errors"
	"runtime"
)

// ErrUnavailable is returned when the app was built without the
// on-device engine.
var ErrUnavailable = errors.New("this build has no on-device engine - rebuild with -tags llama (see docs/building.md), or connect to Ollama / LM Studio on your network")

// LoadOptions configure a model load.
type LoadOptions struct {
	// ContextSize in tokens; 0 picks a default from device memory.
	ContextSize int
	// Threads for generation; 0 uses the performance cores.
	Threads int
	// GPULayers offloads layers to the GPU (Metal on iOS); -1 means all.
	GPULayers int
}

// GenParams are sampling settings.
type GenParams struct {
	Temperature float32
	TopP        float32
	MinP        float32
	TopK        int
	// RepeatPenalty discourages loops; 1 disables it.
	RepeatPenalty float32
	MaxTokens     int
}

// DefaultGenParams suit chat.
func DefaultGenParams() GenParams {
	return GenParams{Temperature: 0.7, TopP: 0.95, MinP: 0.05, TopK: 40, RepeatPenalty: 1.1, MaxTokens: 2048}
}

// TemplateMessage is one message handed to the chat template.
type TemplateMessage struct {
	Role    string
	Content string
}

// Stats describe a generation.
type Stats struct {
	PromptTokens    int
	GeneratedTokens int
	// TokensPerSecond of generation, excluding prompt processing.
	TokensPerSecond float64
	// ReusedTokens were served from the cached prompt prefix.
	ReusedTokens int
}

// Engine is an inference backend holding at most one loaded model.
type Engine interface {
	// Available reports whether the engine is compiled in and can run
	// on this device.
	Available() bool
	// Problem explains why Available is false.
	Problem() string
	// Load loads the GGUF file at path, replacing any loaded model.
	Load(path string, opts LoadOptions) error
	// Loaded returns the path of the loaded model, "" if none.
	Loaded() string
	// Unload frees the model.
	Unload()
	// ContextSize is the loaded context window in tokens.
	ContextSize() int
	// ApplyTemplate formats messages with the model's chat template.
	ApplyTemplate(msgs []TemplateMessage, addAssistant bool) (string, error)
	// CountTokens tokenizes s.
	CountTokens(s string) int
	// Generate continues prompt, calling onPiece for every decoded piece
	// until the model stops, MaxTokens is reached, onPiece returns false
	// or ctx ends.
	Generate(ctx context.Context, prompt string, p GenParams, onPiece func(string) bool) (Stats, error)
}

// DefaultThreads picks a thread count: phones have a few performance
// cores and more efficiency cores, and generation is fastest on the
// former only.
func DefaultThreads() int {
	n := runtime.NumCPU()
	switch {
	case n >= 8:
		return 4
	case n >= 4:
		return n - 2
	default:
		return max(1, n)
	}
}

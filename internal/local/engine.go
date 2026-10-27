// Package local runs language models on the device: GGUF models
// executed by llama.cpp through the engine library in engine/ (downloaded
// and kept up to date by Runtime; linked in on iOS), a catalog of models
// sized for phones and tablets, and a resumable downloader for Hugging
// Face.
package local

import (
	"context"
	"errors"
	"runtime"
)

// ErrUnavailable is returned when no engine is installed.
var ErrUnavailable = errors.New("the on-device engine isn't installed - download it from the model library, or connect to Ollama / LM Studio on your network")

// LoadOptions configure a model load. Zero values pick llama.cpp's
// defaults.
type LoadOptions struct {
	// ContextSize in tokens; 0 picks a default, -1 the model's trained
	// maximum.
	ContextSize int `json:"contextSize,omitempty"`
	// Threads for generation; 0 uses the performance cores.
	Threads int `json:"threads,omitempty"`
	// BatchThreads process the prompt; 0 uses Threads.
	BatchThreads int `json:"batchThreads,omitempty"`
	// GPULayers offloads layers to the GPU (Metal on iOS); -1 means all.
	GPULayers int `json:"gpuLayers,omitempty"`
	// BatchSize and MicroBatchSize split prompt processing; 0 is 512.
	BatchSize      int       `json:"batchSize,omitempty"`
	MicroBatchSize int       `json:"microBatchSize,omitempty"`
	FlashAttn      FlashAttn `json:"flashAttn,omitempty"`
	// CacheTypeK and CacheTypeV store the KV cache; quantized types use
	// less memory (a quantized V cache needs flash attention).
	CacheTypeK KVCacheType `json:"cacheTypeK,omitempty"`
	CacheTypeV KVCacheType `json:"cacheTypeV,omitempty"`
	LoadMode   LoadMode    `json:"loadMode,omitempty"`
	// NoKVOffload keeps the KV cache in CPU memory when layers are on
	// the GPU.
	NoKVOffload bool `json:"noKVOffload,omitempty"`
}

// FlashAttn selects flash attention; "" is FlashAttnAuto.
type FlashAttn string

const (
	FlashAttnAuto FlashAttn = ""
	FlashAttnOn   FlashAttn = "on"
	FlashAttnOff  FlashAttn = "off"
)

// KVCacheType is the KV cache's element type; "" is KVF16.
type KVCacheType string

const (
	KVF16  KVCacheType = ""
	KVBF16 KVCacheType = "bf16"
	KVQ8_0 KVCacheType = "q8_0"
	KVQ5_0 KVCacheType = "q5_0"
	KVQ4_0 KVCacheType = "q4_0"
)

// LoadMode is how model weights are read; "" is LoadAuto.
type LoadMode string

const (
	LoadAuto LoadMode = ""
	// LoadMmap maps the file, letting the OS page weights in and out.
	LoadMmap LoadMode = "mmap"
	// LoadRead reads the whole model into memory.
	LoadRead LoadMode = "read"
	// LoadLock reads it and keeps it resident.
	LoadLock LoadMode = "lock"
)

// GenParams are sampling settings. Zero values switch a sampler off;
// DefaultGenParams suit chat.
type GenParams struct {
	// Temperature 0 picks the most likely token every time.
	Temperature float32 `json:"temperature"`
	TopK        int     `json:"topK"`
	TopP        float32 `json:"topP"`
	MinP        float32 `json:"minP"`
	TypicalP    float32 `json:"typicalP"`
	TopNSigma   float32 `json:"topNSigma"`
	// RepeatPenalty discourages loops; 1 disables it.
	RepeatPenalty    float32 `json:"repeatPenalty"`
	FrequencyPenalty float32 `json:"frequencyPenalty"`
	PresencePenalty  float32 `json:"presencePenalty"`
	// RepeatLastN is how far back the penalties look; -1 is the whole
	// context.
	RepeatLastN int `json:"repeatLastN"`
	// DRY ("don't repeat yourself") penalizes repeating sequences;
	// multiplier 0 disables it.
	DRYMultiplier    float32 `json:"dryMultiplier"`
	DRYBase          float32 `json:"dryBase"`
	DRYAllowedLength int     `json:"dryAllowedLength"`
	// XTC ("exclude top choices") sometimes drops the likeliest tokens
	// for more varied text; probability 0 disables it.
	XTCProbability float32 `json:"xtcProbability"`
	XTCThreshold   float32 `json:"xtcThreshold"`
	// Mirostat 2 targets a perplexity (tau) instead of truncating.
	Mirostat    int     `json:"mirostat"`
	MirostatTau float32 `json:"mirostatTau"`
	MirostatEta float32 `json:"mirostatEta"`
	// Seed makes sampling repeatable; 0 is random.
	Seed      uint32 `json:"seed"`
	MaxTokens int    `json:"maxTokens"`
}

// DefaultGenParams suit chat.
func DefaultGenParams() GenParams {
	return GenParams{
		Temperature: 0.7, TopP: 0.95, MinP: 0.05, TopK: 40, TypicalP: 1,
		RepeatPenalty: 1.1, RepeatLastN: 64,
		DRYBase: 1.75, DRYAllowedLength: 2, XTCThreshold: 0.1,
		MirostatTau: 5, MirostatEta: 0.1,
		MaxTokens: 2048,
	}
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

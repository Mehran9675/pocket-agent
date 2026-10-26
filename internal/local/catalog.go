package local

import "fmt"

// CatalogModel is a model Pocket Agent recommends for phones and
// tablets: small, instruction-tuned, quantized to Q4_K_M (or Q8 for
// the tiniest ones), with a permissive license.
type CatalogModel struct {
	ID          string
	Name        string
	Vendor      string
	Params      string
	Quant       string
	Description string
	License     string
	// Repo and File locate the GGUF on Hugging Face.
	Repo string
	File string
	// Size is the download size in bytes (approximate).
	Size int64
	// MinRAM is the device memory, in GB, the model runs comfortably in.
	MinRAM    float64
	Context   int
	Reasoning bool
	Tools     bool
}

// URL is the download link.
func (m CatalogModel) URL() string {
	return HFResolveURL(m.Repo, m.File)
}

// HFResolveURL is the download URL of a file in a Hugging Face repo.
func HFResolveURL(repo, file string) string {
	return fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s?download=true", repo, file)
}

const gb = 1 << 30

// Catalog lists the recommended models, smallest first.
var Catalog = []CatalogModel{
	{
		ID: "qwen3-0.6b", Name: "Qwen3 0.6B", Vendor: "Alibaba", Params: "0.6B", Quant: "Q8_0",
		Description: "Tiny and fast. Thinks before answering; good for quick questions on any phone.",
		License:     "Apache 2.0", Repo: "Qwen/Qwen3-0.6B-GGUF", File: "Qwen3-0.6B-Q8_0.gguf",
		Size: 639 << 20, MinRAM: 3, Context: 32768, Reasoning: true, Tools: true,
	},
	{
		ID: "llama-3.2-1b", Name: "Llama 3.2 1B Instruct", Vendor: "Meta", Params: "1B", Quant: "Q4_K_M",
		Description: "Meta's compact assistant. Quick, multilingual, good at summaries and rewriting.",
		License:     "Llama 3.2 Community", Repo: "bartowski/Llama-3.2-1B-Instruct-GGUF", File: "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
		Size: 808 << 20, MinRAM: 3, Context: 131072, Tools: true,
	},
	{
		ID: "gemma-3-1b", Name: "Gemma 3 1B", Vendor: "Google", Params: "1B", Quant: "Q4_K_M",
		Description: "Google's smallest Gemma. Fluent writing at a tiny size.",
		License:     "Gemma", Repo: "ggml-org/gemma-3-1b-it-GGUF", File: "gemma-3-1b-it-Q4_K_M.gguf",
		Size: 806 << 20, MinRAM: 3, Context: 32768,
	},
	{
		ID: "qwen3-1.7b", Name: "Qwen3 1.7B", Vendor: "Alibaba", Params: "1.7B", Quant: "Q4_K_M",
		Description: "The best balance for most phones: reasoning, tool use and code.",
		License:     "Apache 2.0", Repo: "unsloth/Qwen3-1.7B-GGUF", File: "Qwen3-1.7B-Q4_K_M.gguf",
		Size: 1107 << 20, MinRAM: 4, Context: 32768, Reasoning: true, Tools: true,
	},
	{
		ID: "smollm2-1.7b", Name: "SmolLM2 1.7B Instruct", Vendor: "Hugging Face", Params: "1.7B", Quant: "Q4_K_M",
		Description: "Fully open model trained on curated data. Efficient general assistant.",
		License:     "Apache 2.0", Repo: "bartowski/SmolLM2-1.7B-Instruct-GGUF", File: "SmolLM2-1.7B-Instruct-Q4_K_M.gguf",
		Size: 1060 << 20, MinRAM: 4, Context: 8192,
	},
	{
		ID: "llama-3.2-3b", Name: "Llama 3.2 3B Instruct", Vendor: "Meta", Params: "3B", Quant: "Q4_K_M",
		Description: "Stronger reasoning and instruction following than the 1B, still phone-friendly.",
		License:     "Llama 3.2 Community", Repo: "bartowski/Llama-3.2-3B-Instruct-GGUF", File: "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		Size: 2019 << 20, MinRAM: 6, Context: 131072, Tools: true,
	},
	{
		ID: "phi-4-mini", Name: "Phi-4 mini Instruct", Vendor: "Microsoft", Params: "3.8B", Quant: "Q4_K_M",
		Description: "Excellent at math and logic for its size.",
		License:     "MIT", Repo: "bartowski/microsoft_Phi-4-mini-instruct-GGUF", File: "microsoft_Phi-4-mini-instruct-Q4_K_M.gguf",
		Size: 2491 << 20, MinRAM: 6, Context: 131072, Tools: true,
	},
	{
		ID: "qwen3-4b", Name: "Qwen3 4B", Vendor: "Alibaba", Params: "4B", Quant: "Q4_K_M",
		Description: "The most capable choice for flagship phones and iPads: reasoning, tools and code.",
		License:     "Apache 2.0", Repo: "unsloth/Qwen3-4B-GGUF", File: "Qwen3-4B-Q4_K_M.gguf",
		Size: 2497 << 20, MinRAM: 8, Context: 32768, Reasoning: true, Tools: true,
	},
	{
		ID: "gemma-3-4b", Name: "Gemma 3 4B", Vendor: "Google", Params: "4B", Quant: "Q4_K_M",
		Description: "High-quality writing and broad knowledge across 140 languages.",
		License:     "Gemma", Repo: "unsloth/gemma-3-4b-it-GGUF", File: "gemma-3-4b-it-Q4_K_M.gguf",
		Size: 2489 << 20, MinRAM: 8, Context: 131072,
	},
}

// CatalogByFile finds the catalog entry for a GGUF file name.
func CatalogByFile(file string) (CatalogModel, bool) {
	for _, m := range Catalog {
		if m.File == file {
			return m, true
		}
	}
	return CatalogModel{}, false
}

// Fit rates how well a model suits a device with ram bytes of memory.
type Fit int

const (
	FitUnknown Fit = iota
	FitGreat
	FitOK
	FitTight
)

// FitFor rates m against ram (0 when unknown).
func FitFor(minRAM float64, ram uint64) Fit {
	if ram == 0 {
		return FitUnknown
	}
	have := float64(ram) / gb
	switch {
	case have >= minRAM*1.5:
		return FitGreat
	case have >= minRAM:
		return FitOK
	default:
		return FitTight
	}
}

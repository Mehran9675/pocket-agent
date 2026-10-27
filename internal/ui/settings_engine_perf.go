package ui

import (
	"runtime"
	"slices"
	"strconv"

	"github.com/mehran9675/giode"
	"github.com/mehran9675/pocket-agent/internal/local"
	"github.com/mehran9675/pocket-agent/internal/ui/theme"
	"github.com/mehran9675/pocket-agent/internal/ui/widgets"
)

var (
	kvTypes      = []local.KVCacheType{local.KVF16, local.KVBF16, local.KVQ8_0, local.KVQ5_0, local.KVQ4_0}
	kvLabels     = []string{"F16", "BF16", "Q8_0", "Q5_0", "Q4_0"}
	flashModes   = []local.FlashAttn{local.FlashAttnAuto, local.FlashAttnOn, local.FlashAttnOff}
	flashLabels  = []string{"Auto", "On", "Off"}
	loadModes    = []local.LoadMode{local.LoadAuto, local.LoadMmap, local.LoadRead, local.LoadLock}
	loadLabels   = []string{"Auto", "Memory-map", "Read into RAM", "Lock in RAM"}
	batchSizes   = []int{64, 128, 256, 512, 1024, 2048}
	batchLabels  = []string{"64", "128", "256", "512", "1024", "2048"}
	threadLabels = func() []string {
		l := []string{"Auto"}
		for i := 1; i <= runtime.NumCPU(); i++ {
			l = append(l, strconv.Itoa(i))
		}
		return l
	}()
)

// indexOr is the index of v in list, or def.
func indexOr[T comparable](list []T, v T, def int) int {
	if i := slices.Index(list, v); i >= 0 {
		return i
	}
	return def
}

// enginePerfBody tunes how models load and run: threads, batches,
// flash attention, the KV cache and memory.
func (a *App) enginePerfBody() widgets.El {
	ls := &a.settings.Local
	lo := &ls.Load
	// changed saves a load option; it applies when the model next loads.
	changed := func() {
		a.saveSettings()
		a.engine.Unload()
	}
	batch := lo.BatchSize
	if batch == 0 {
		batch = 512
	}
	ubatch := lo.MicroBatchSize
	if ubatch == 0 {
		ubatch = 512
	}
	return widgets.Col(widgets.St{Gap: 16, Width: -1},
		widgets.Text("Changes apply the next time a model loads. Defaults suit most phones; these are for squeezing in a larger model or context, or more speed.", theme.SizeFootnote, theme.T.Muted),
		widgets.SectionLabel("Memory"),
		widgets.Card(widgets.St{Gap: 16},
			widgets.SelectRow("perf/k", "K cache type", "", kvLabels, indexOr(kvTypes, lo.CacheTypeK, 0), func(i int) {
				lo.CacheTypeK = kvTypes[i]
				changed()
			}),
			widgets.SelectRow("perf/v", "V cache type", "The KV cache holds the conversation so far and grows with the context window. Q8_0 halves its memory with little loss; Q4_0 quarters it. A quantized V cache needs flash attention.", kvLabels, indexOr(kvTypes, lo.CacheTypeV, 0), func(i int) {
				lo.CacheTypeV = kvTypes[i]
				if lo.CacheTypeV != local.KVF16 && lo.CacheTypeV != local.KVBF16 && lo.FlashAttn == local.FlashAttnOff {
					lo.FlashAttn = local.FlashAttnAuto
				}
				changed()
			}),
			widgets.SelectRow("perf/fa", "Flash attention", "Faster, leaner attention. Auto uses it where the model and device support it.", flashLabels, indexOr(flashModes, lo.FlashAttn, 0), func(i int) {
				lo.FlashAttn = flashModes[i]
				changed()
			}),
			widgets.SelectRow("perf/load", "Model loading", "Memory-map starts fastest and lets the system reclaim memory; reading into RAM avoids stalls when memory is tight; locking keeps the model resident.", loadLabels, indexOr(loadModes, lo.LoadMode, 0), func(i int) {
				lo.LoadMode = loadModes[i]
				changed()
			}),
		),
		widgets.SectionLabel("Speed"),
		widgets.Card(widgets.St{Gap: 16},
			widgets.SelectRow("perf/threads", "Generation threads", "", threadLabels, min(ls.Threads, len(threadLabels)-1), func(i int) {
				ls.Threads = i
				changed()
			}),
			widgets.SelectRow("perf/bthreads", "Prompt threads", "Auto uses the performance cores. More threads than performance cores is usually slower on phones.", threadLabels, min(lo.BatchThreads, len(threadLabels)-1), func(i int) {
				lo.BatchThreads = i
				changed()
			}),
			widgets.SelectRow("perf/batch", "Batch size", "", batchLabels, indexOr(batchSizes, batch, 3), func(i int) {
				lo.BatchSize = batchSizes[i]
				if lo.MicroBatchSize > lo.BatchSize {
					lo.MicroBatchSize = lo.BatchSize
				}
				changed()
			}),
			widgets.SelectRow("perf/ubatch", "Micro-batch size", "How much of a prompt is processed at once. Larger is faster to the first token but needs more memory.", batchLabels, indexOr(batchSizes, ubatch, 3), func(i int) {
				lo.MicroBatchSize = batchSizes[i]
				if lo.BatchSize != 0 && lo.MicroBatchSize > lo.BatchSize {
					lo.BatchSize = lo.MicroBatchSize
				}
				changed()
			}),
		),
		widgets.SectionLabel("GPU"),
		widgets.Group(
			widgets.ListRow("perf/gpu", "zap", "#34d399", "GPU acceleration", "Run the model on the GPU (Metal on iPhone and iPad)", widgets.Toggle("perf/gpu/t", ls.GPULayers != 0, func(v bool) {
				ls.GPULayers = ifInt(v, -1, 0)
				changed()
			}), nil),
			widgets.ListRow("perf/kqv", "layers", "#34d399", "KV cache on the GPU", "Faster with GPU acceleration; off saves GPU memory", widgets.Toggle("perf/kqv/t", !lo.NoKVOffload, func(v bool) {
				lo.NoKVOffload = !v
				changed()
			}), nil),
		),
		widgets.Button("perf/reset", "Reset to defaults", "rotate-ccw", widgets.BtnSecondary, func() {
			*lo = local.LoadOptions{}
			ls.Threads, ls.GPULayers = 0, -1
			changed()
			a.toast(giode.ToastInfo, "Engine settings reset")
		}, widgets.St{Width: -1}),
	)
}

func ifInt(c bool, a, b int) int {
	if c {
		return a
	}
	return b
}

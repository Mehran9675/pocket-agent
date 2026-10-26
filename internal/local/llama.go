//go:build llama && cgo

package local

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/llama.cpp/include -I${SRCDIR}/../../third_party/llama.cpp/ggml/include -O2
#cgo linux,!android LDFLAGS: -L${SRCDIR}/../../third_party/lib/linux -lllama -lggml -lggml-cpu -lggml-base -lstdc++ -lm -lpthread
#cgo android,arm64 LDFLAGS: -L${SRCDIR}/../../third_party/lib/android_arm64 -lllama -lggml -lggml-cpu -lggml-base -lc++_static -lc++abi -lm -landroid -llog
#cgo android,amd64 LDFLAGS: -L${SRCDIR}/../../third_party/lib/android_amd64 -lllama -lggml -lggml-cpu -lggml-base -lc++_static -lc++abi -lm -landroid -llog
#cgo ios,arm64 LDFLAGS: -L${SRCDIR}/../../third_party/lib/ios_arm64 -lllama -lggml -lggml-cpu -lggml-metal -lggml-blas -lggml-base -lc++ -framework Foundation -framework Metal -framework MetalKit -framework Accelerate
#cgo darwin,!ios LDFLAGS: -L${SRCDIR}/../../third_party/lib/darwin -lllama -lggml -lggml-cpu -lggml-metal -lggml-blas -lggml-base -lc++ -framework Foundation -framework Metal -framework MetalKit -framework Accelerate

#include <stdlib.h>
#include <string.h>
#include "llama.h"

static bool pa_abort(void * data) { return *(volatile int *)data != 0; }

static void pa_set_abort(struct llama_context_params * p, int * flag) {
	p->abort_callback = pa_abort;
	p->abort_callback_data = flag;
}

static void pa_quiet(enum ggml_log_level level, const char * text, void * data) {}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

var backendOnce sync.Once

type llamaEngine struct {
	mu    sync.Mutex
	path  string
	model *C.struct_llama_model
	ctx   *C.struct_llama_context
	vocab *C.struct_llama_vocab
	nctx  int
	// abort is C memory the abort callback polls.
	abort *C.int
	// cached are the tokens currently in the KV cache, so a follow-up
	// turn only decodes what's new.
	cached []C.llama_token
}

// NewEngine returns the llama.cpp engine.
func NewEngine() Engine {
	backendOnce.Do(func() {
		C.llama_log_set(C.ggml_log_callback(C.pa_quiet), nil)
		C.llama_backend_init()
	})
	e := &llamaEngine{abort: (*C.int)(C.calloc(1, C.size_t(unsafe.Sizeof(C.int(0)))))}
	return e
}

func (e *llamaEngine) Available() bool { return true }

func (e *llamaEngine) Loaded() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.path
}

func (e *llamaEngine) ContextSize() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.nctx
}

func (e *llamaEngine) Load(path string, opts LoadOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.path == path && e.model != nil {
		return nil
	}
	e.unloadLocked()

	mp := C.llama_model_default_params()
	mp.n_gpu_layers = C.int32_t(opts.GPULayers)
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	model := C.llama_model_load_from_file(cpath, mp)
	if model == nil {
		return fmt.Errorf("couldn't load %s - the file may be damaged or use an architecture this engine doesn't support", path)
	}

	cp := C.llama_context_default_params()
	nctx := opts.ContextSize
	if train := int(C.llama_model_n_ctx_train(model)); nctx <= 0 || (train > 0 && nctx > train) {
		if nctx <= 0 {
			nctx = 4096
		}
		if train > 0 && nctx > train {
			nctx = train
		}
	}
	cp.n_ctx = C.uint32_t(nctx)
	cp.n_batch = 512
	cp.n_ubatch = 512
	threads := opts.Threads
	if threads <= 0 {
		threads = DefaultThreads()
	}
	cp.n_threads = C.int32_t(threads)
	cp.n_threads_batch = C.int32_t(threads)
	cp.no_perf = true
	C.pa_set_abort(&cp, e.abort)
	ctx := C.llama_init_from_model(model, cp)
	if ctx == nil {
		C.llama_model_free(model)
		return errors.New("not enough memory for this model's context - pick a smaller model or context size")
	}
	e.model, e.ctx = model, ctx
	e.vocab = C.llama_model_get_vocab(model)
	e.nctx = int(C.llama_n_ctx(ctx))
	e.path = path
	e.cached = nil
	return nil
}

func (e *llamaEngine) Unload() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.unloadLocked()
}

func (e *llamaEngine) unloadLocked() {
	if e.ctx != nil {
		C.llama_free(e.ctx)
		e.ctx = nil
	}
	if e.model != nil {
		C.llama_model_free(e.model)
		e.model = nil
	}
	e.path, e.nctx, e.cached, e.vocab = "", 0, nil, nil
}

func (e *llamaEngine) ApplyTemplate(msgs []TemplateMessage, addAssistant bool) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.model == nil {
		return "", errors.New("no model loaded")
	}
	tmpl := C.llama_model_chat_template(e.model, nil)
	cmsgs := make([]C.struct_llama_chat_message, len(msgs))
	var frees []unsafe.Pointer
	defer func() {
		for _, p := range frees {
			C.free(p)
		}
	}()
	total := 0
	for i, m := range msgs {
		role, content := C.CString(m.Role), C.CString(m.Content)
		frees = append(frees, unsafe.Pointer(role), unsafe.Pointer(content))
		cmsgs[i].role, cmsgs[i].content = role, content
		total += len(m.Content) + 64
	}
	if len(cmsgs) == 0 {
		return "", nil
	}
	size := total*2 + 1024
	for range 2 {
		buf := (*C.char)(C.malloc(C.size_t(size)))
		n := C.llama_chat_apply_template(tmpl, &cmsgs[0], C.size_t(len(cmsgs)), C.bool(addAssistant), buf, C.int32_t(size))
		if n < 0 {
			C.free(unsafe.Pointer(buf))
			// No usable template in the file: fall back to ChatML,
			// which most instruction-tuned models understand.
			return chatML(msgs, addAssistant), nil
		}
		if int(n) <= size {
			out := C.GoStringN(buf, n)
			C.free(unsafe.Pointer(buf))
			return out, nil
		}
		C.free(unsafe.Pointer(buf))
		size = int(n) + 1
	}
	return chatML(msgs, addAssistant), nil
}

func (e *llamaEngine) tokenize(s string, addSpecial bool) []C.llama_token {
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	n := len(s) + 16
	toks := make([]C.llama_token, n)
	got := C.llama_tokenize(e.vocab, cs, C.int32_t(len(s)), &toks[0], C.int32_t(n), C.bool(addSpecial), true)
	if got < 0 {
		toks = make([]C.llama_token, -got)
		got = C.llama_tokenize(e.vocab, cs, C.int32_t(len(s)), &toks[0], C.int32_t(-got), C.bool(addSpecial), true)
	}
	if got < 0 {
		return nil
	}
	return toks[:got]
}

func (e *llamaEngine) CountTokens(s string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vocab == nil {
		return len(s) / 4
	}
	return len(e.tokenize(s, false))
}

func (e *llamaEngine) piece(tok C.llama_token) string {
	var buf [256]C.char
	n := C.llama_token_to_piece(e.vocab, tok, &buf[0], C.int32_t(len(buf)), 0, false)
	if n < 0 {
		big := make([]C.char, -n)
		n = C.llama_token_to_piece(e.vocab, tok, &big[0], C.int32_t(len(big)), 0, false)
		if n < 0 {
			return ""
		}
		return C.GoStringN(&big[0], n)
	}
	return C.GoStringN(&buf[0], n)
}

func (e *llamaEngine) decode(toks []C.llama_token) error {
	const batch = 512
	for i := 0; i < len(toks); i += batch {
		j := min(i+batch, len(toks))
		chunk := toks[i:j]
		if rc := C.llama_decode(e.ctx, C.llama_batch_get_one(&chunk[0], C.int32_t(len(chunk)))); rc != 0 {
			if rc == 2 {
				return context.Canceled
			}
			return fmt.Errorf("decode failed (%d)", int(rc))
		}
	}
	return nil
}

func (e *llamaEngine) Generate(ctx context.Context, prompt string, p GenParams, onPiece func(string) bool) (Stats, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var st Stats
	if e.ctx == nil {
		return st, errors.New("no model loaded")
	}
	*e.abort = 0
	stop := context.AfterFunc(ctx, func() { *e.abort = 1 })
	defer stop()

	toks := e.tokenize(prompt, true)
	if len(toks) == 0 {
		return st, errors.New("empty prompt")
	}
	st.PromptTokens = len(toks)
	if len(toks) >= e.nctx-8 {
		return st, fmt.Errorf("the conversation (%d tokens) no longer fits this model's %d-token context - start a new chat or raise the context size", len(toks), e.nctx)
	}

	// Reuse the longest cached prefix; at least one token must be
	// decoded to get logits.
	keep := 0
	for keep < len(e.cached) && keep < len(toks) && e.cached[keep] == toks[keep] {
		keep++
	}
	if keep == len(toks) {
		keep--
	}
	mem := C.llama_get_memory(e.ctx)
	if !bool(C.llama_memory_seq_rm(mem, 0, C.llama_pos(keep), -1)) {
		C.llama_memory_clear(mem, true)
		keep = 0
	}
	e.cached = append(e.cached[:0:0], toks[:keep]...)
	st.ReusedTokens = keep
	if err := e.decode(toks[keep:]); err != nil {
		e.cached = nil
		C.llama_memory_clear(mem, true)
		return st, err
	}
	e.cached = append(e.cached, toks[keep:]...)

	smpl := e.sampler(p)
	defer C.llama_sampler_free(smpl)

	maxTokens := p.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	start := time.Now()
	var pending []byte // bytes of an incomplete UTF-8 sequence
	for st.GeneratedTokens < maxTokens {
		if ctx.Err() != nil {
			return st, ctx.Err()
		}
		if len(e.cached) >= e.nctx-1 {
			break
		}
		tok := C.llama_sampler_sample(smpl, e.ctx, -1)
		if bool(C.llama_vocab_is_eog(e.vocab, tok)) {
			break
		}
		st.GeneratedTokens++
		pending = append(pending, e.piece(tok)...)
		text, rest := splitValidUTF8(pending)
		pending = rest
		if text != "" && !onPiece(text) {
			break
		}
		one := []C.llama_token{tok}
		if rc := C.llama_decode(e.ctx, C.llama_batch_get_one(&one[0], 1)); rc != 0 {
			if rc == 2 {
				return st, context.Canceled
			}
			return st, fmt.Errorf("decode failed (%d)", int(rc))
		}
		e.cached = append(e.cached, tok)
	}
	if len(pending) > 0 {
		onPiece(strings.ToValidUTF8(string(pending), ""))
	}
	if d := time.Since(start).Seconds(); d > 0 {
		st.TokensPerSecond = float64(st.GeneratedTokens) / d
	}
	return st, nil
}

func (e *llamaEngine) sampler(p GenParams) *C.struct_llama_sampler {
	chain := C.llama_sampler_chain_init(C.llama_sampler_chain_default_params())
	if p.RepeatPenalty > 0 && p.RepeatPenalty != 1 {
		C.llama_sampler_chain_add(chain, C.llama_sampler_init_penalties(C.llama_vocab_n_tokens(e.vocab), 64, C.float(p.RepeatPenalty), 0, 0))
	}
	if p.Temperature <= 0 {
		C.llama_sampler_chain_add(chain, C.llama_sampler_init_greedy())
		return chain
	}
	if p.TopK > 0 {
		C.llama_sampler_chain_add(chain, C.llama_sampler_init_top_k(C.int32_t(p.TopK)))
	}
	if p.TopP > 0 && p.TopP < 1 {
		C.llama_sampler_chain_add(chain, C.llama_sampler_init_top_p(C.float(p.TopP), 1))
	}
	if p.MinP > 0 {
		C.llama_sampler_chain_add(chain, C.llama_sampler_init_min_p(C.float(p.MinP), 1))
	}
	C.llama_sampler_chain_add(chain, C.llama_sampler_init_temp(C.float(p.Temperature)))
	C.llama_sampler_chain_add(chain, C.llama_sampler_init_dist(C.LLAMA_DEFAULT_SEED))
	return chain
}

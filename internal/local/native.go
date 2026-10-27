//go:build cgo

package local

/*
#cgo CFLAGS: -I${SRCDIR}/../../engine
#cgo linux LDFLAGS: -ldl
#include <stdlib.h>
#include "bridge.h"

extern int32_t paGoPiece(void *, char *, int32_t);

static int32_t w_abi(void) { return pa_api.abi_version(); }
static const char *w_version(void) { return pa_api.engine_version(); }
static pa_model *w_load(const char *path, int32_t n_ctx, int32_t threads, int32_t gpu, int32_t *abort, char *err, int32_t errlen) {
	return pa_api.load(path, n_ctx, threads, gpu, abort, err, errlen);
}
static void w_free(pa_model *m) { pa_api.free(m); }
static int32_t w_n_ctx(pa_model *m) { return pa_api.n_ctx(m); }
static int32_t w_template(pa_model *m, const pa_message *msgs, int32_t n, int32_t add, char *buf, int32_t len) {
	return pa_api.apply_template(m, msgs, n, add, buf, len);
}
static int32_t w_count(pa_model *m, const char *s, int32_t len) { return pa_api.count_tokens(m, s, len); }
static int32_t w_generate(pa_model *m, const char *prompt, int32_t len, const pa_params *p, uintptr_t user, pa_stats *st, char *err, int32_t errlen) {
	return pa_api.generate(m, prompt, len, p, (pa_piece_fn)paGoPiece, (void *)user, st, err, errlen);
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"runtime/cgo"
	"strings"
	"sync"
	"unsafe"
)

// engineABI is the PA_ABI_VERSION of engine/pocketllama.h this binding
// speaks; Runtime only installs engines built for it.
const engineABI = C.PA_ABI_VERSION

const errLen = 512

// The engine library is bound once per process: a loaded library can't
// be swapped safely, so an update takes effect on the next launch.
var (
	bindMu       sync.Mutex
	boundVersion string
)

// bindEngine loads the engine library at lib (ignored when it's linked
// in) and returns its version.
func bindEngine(lib string) (string, error) {
	bindMu.Lock()
	defer bindMu.Unlock()
	if boundVersion != "" {
		return boundVersion, nil
	}
	clib := C.CString(lib)
	defer C.free(unsafe.Pointer(clib))
	cerr := (*C.char)(C.calloc(errLen, 1))
	defer C.free(unsafe.Pointer(cerr))
	if C.pa_bind(clib, cerr, errLen) != 0 {
		return "", fmt.Errorf("couldn't load the on-device engine: %s", C.GoString(cerr))
	}
	if abi := int(C.w_abi()); abi != engineABI {
		return "", fmt.Errorf("the installed engine (interface %d) doesn't match this app (interface %d) - update the engine or the app", abi, engineABI)
	}
	boundVersion = C.GoString(C.w_version())
	return boundVersion, nil
}

type nativeEngine struct {
	rt *Runtime

	mu    sync.Mutex
	path  string
	model *C.pa_model
	nctx  int
	// abort is C memory the engine polls while decoding.
	abort *C.int32_t
}

// NewEngine returns the llama.cpp engine, which runs once rt has an
// engine installed (or the app has it built in). It touches no native
// code until a model is loaded.
func NewEngine(rt *Runtime) Engine {
	return &nativeEngine{rt: rt}
}

func (e *nativeEngine) Available() bool { return e.rt.Ready() }

func (e *nativeEngine) Problem() string { return e.rt.Problem() }

func (e *nativeEngine) Loaded() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.path
}

func (e *nativeEngine) ContextSize() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.nctx
}

func (e *nativeEngine) Load(path string, opts LoadOptions) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.path == path && e.model != nil {
		return nil
	}
	lib := e.rt.LibraryPath()
	if lib == "" && !builtinEngine {
		return ErrUnavailable
	}
	version, err := bindEngine(lib)
	if err != nil {
		return err
	}
	e.rt.setLoaded(version)
	e.unloadLocked()
	if e.abort == nil {
		e.abort = (*C.int32_t)(C.calloc(1, C.size_t(unsafe.Sizeof(C.int32_t(0)))))
	}
	threads := opts.Threads
	if threads <= 0 {
		threads = DefaultThreads()
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	cerr := (*C.char)(C.calloc(errLen, 1))
	defer C.free(unsafe.Pointer(cerr))
	m := C.w_load(cpath, C.int32_t(opts.ContextSize), C.int32_t(threads), C.int32_t(opts.GPULayers), e.abort, cerr, errLen)
	if m == nil {
		return errors.New(C.GoString(cerr))
	}
	e.model, e.nctx, e.path = m, int(C.w_n_ctx(m)), path
	return nil
}

func (e *nativeEngine) Unload() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.unloadLocked()
}

func (e *nativeEngine) unloadLocked() {
	if e.model != nil {
		C.w_free(e.model)
		e.model = nil
	}
	e.path, e.nctx = "", 0
}

func (e *nativeEngine) ApplyTemplate(msgs []TemplateMessage, addAssistant bool) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.model == nil {
		return "", errors.New("no model loaded")
	}
	if len(msgs) == 0 {
		return "", nil
	}
	cmsgs := (*C.pa_message)(C.malloc(C.size_t(len(msgs)) * C.size_t(unsafe.Sizeof(C.pa_message{}))))
	defer C.free(unsafe.Pointer(cmsgs))
	list := unsafe.Slice(cmsgs, len(msgs))
	total := 0
	for i, m := range msgs {
		list[i].role, list[i].content = C.CString(m.Role), C.CString(m.Content)
		total += len(m.Content) + 64
	}
	defer func() {
		for i := range list {
			C.free(unsafe.Pointer(list[i].role))
			C.free(unsafe.Pointer(list[i].content))
		}
	}()
	add := C.int32_t(0)
	if addAssistant {
		add = 1
	}
	size := total*2 + 1024
	for range 2 {
		buf := (*C.char)(C.malloc(C.size_t(size)))
		n := int(C.w_template(e.model, cmsgs, C.int32_t(len(msgs)), add, buf, C.int32_t(size)))
		if n < 0 {
			C.free(unsafe.Pointer(buf))
			// No usable template in the file: fall back to ChatML,
			// which most instruction-tuned models understand.
			return chatML(msgs, addAssistant), nil
		}
		if n <= size {
			out := C.GoStringN(buf, C.int(n))
			C.free(unsafe.Pointer(buf))
			return out, nil
		}
		C.free(unsafe.Pointer(buf))
		size = n + 1
	}
	return chatML(msgs, addAssistant), nil
}

func (e *nativeEngine) CountTokens(s string) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.model == nil {
		return len(s) / 4
	}
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	return int(C.w_count(e.model, cs, C.int32_t(len(s))))
}

// generation is the state of one Generate call, reached from the
// engine's piece callback through a cgo.Handle.
type generation struct {
	onPiece func(string) bool
	// pending holds the bytes of an incomplete UTF-8 sequence.
	pending []byte
}

func (g *generation) piece(b []byte) bool {
	g.pending = append(g.pending, b...)
	text, rest := splitValidUTF8(g.pending)
	g.pending = rest
	return text == "" || g.onPiece(text)
}

func (e *nativeEngine) Generate(ctx context.Context, prompt string, p GenParams, onPiece func(string) bool) (Stats, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	var st Stats
	if e.model == nil {
		return st, errors.New("no model loaded")
	}
	*e.abort = 0
	stop := context.AfterFunc(ctx, func() { *e.abort = 1 })
	defer stop()

	g := &generation{onPiece: onPiece}
	h := cgo.NewHandle(g)
	defer h.Delete()
	params := C.pa_params{
		temperature:    C.float(p.Temperature),
		top_p:          C.float(p.TopP),
		min_p:          C.float(p.MinP),
		repeat_penalty: C.float(p.RepeatPenalty),
		top_k:          C.int32_t(p.TopK),
		max_tokens:     C.int32_t(p.MaxTokens),
	}
	cprompt := C.CString(prompt)
	defer C.free(unsafe.Pointer(cprompt))
	cerr := (*C.char)(C.calloc(errLen, 1))
	defer C.free(unsafe.Pointer(cerr))
	var cst C.pa_stats
	rc := C.w_generate(e.model, cprompt, C.int32_t(len(prompt)), &params, C.uintptr_t(h), &cst, cerr, errLen)
	if len(g.pending) > 0 {
		onPiece(strings.ToValidUTF8(string(g.pending), ""))
	}
	st = Stats{PromptTokens: int(cst.prompt_tokens), GeneratedTokens: int(cst.generated_tokens), ReusedTokens: int(cst.reused_tokens)}
	if secs := float64(cst.seconds); secs > 0 {
		st.TokensPerSecond = float64(st.GeneratedTokens) / secs
	}
	switch rc {
	case C.PA_OK:
		return st, nil
	case C.PA_CANCELED:
		if ctx.Err() != nil {
			return st, ctx.Err()
		}
		return st, context.Canceled
	case C.PA_TOO_LONG:
		return st, fmt.Errorf("the conversation (%d tokens) no longer fits this model's %d-token context - start a new chat or raise the context size", st.PromptTokens, e.nctx)
	default:
		return st, errors.New(C.GoString(cerr))
	}
}

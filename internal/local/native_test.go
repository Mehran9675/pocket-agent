//go:build cgo

package local

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestNativeEngine runs a real engine build against a model:
//
//	PA_TEST_ENGINE=build/engine-linux/libpocketllama.so PA_TEST_MODEL=tiny.gguf go test ./internal/local -run Native
func TestNativeEngine(t *testing.T) {
	libPath, model := os.Getenv("PA_TEST_ENGINE"), os.Getenv("PA_TEST_MODEL")
	if libPath == "" || model == "" {
		t.Skip("set PA_TEST_ENGINE and PA_TEST_MODEL")
	}
	lib, err := os.ReadFile(libPath)
	if err != nil {
		t.Fatal(err)
	}
	version := "test"
	srv := engineServer(t, lib, &version, false)
	rt := NewRuntime(t.TempDir(), srv.URL+"/engine-manifest.json")
	if err := rt.Install(context.Background()); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(rt)
	if !e.Available() {
		t.Fatal(e.Problem())
	}
	if err := e.Load(model, LoadOptions{ContextSize: 256}); err != nil {
		t.Fatal(err)
	}
	defer e.Unload()
	if e.ContextSize() != 256 || rt.Status().Loaded == "" {
		t.Fatalf("context %d, loaded %q", e.ContextSize(), rt.Status().Loaded)
	}
	prompt, err := e.ApplyTemplate([]TemplateMessage{{Role: "user", Content: "Once upon a time"}}, true)
	if err != nil || !strings.Contains(prompt, "Once upon a time") {
		t.Fatalf("template %q, %v", prompt, err)
	}
	if n := e.CountTokens("Once upon a time"); n <= 0 {
		t.Fatalf("count %d", n)
	}
	p := DefaultGenParams()
	p.MaxTokens = 12
	var out strings.Builder
	st, err := e.Generate(context.Background(), prompt, p, func(s string) bool { out.WriteString(s); return true })
	if err != nil || st.GeneratedTokens == 0 || st.PromptTokens == 0 {
		t.Fatalf("generate: %+v %v", st, err)
	}
	t.Logf("%d tokens at %.0f tok/s: %q", st.GeneratedTokens, st.TokensPerSecond, out.String())

	// The follow-up reuses the cached prompt.
	st, err = e.Generate(context.Background(), prompt+out.String()+"\n<|user|>more", p, func(string) bool { return true })
	if err != nil || st.ReusedTokens == 0 {
		t.Fatalf("follow-up: %+v %v", st, err)
	}

	// Cancelation stops decoding.
	ctx, cancel := context.WithCancel(context.Background())
	n := 0
	p.MaxTokens = 200
	_, err = e.Generate(ctx, prompt, p, func(string) bool {
		if n++; n == 3 {
			cancel()
		}
		return true
	})
	if err != context.Canceled {
		t.Fatalf("canceled generate returned %v", err)
	}

	// Too long a prompt is reported, not crashed on.
	if _, err := e.Generate(context.Background(), strings.Repeat("word ", 400), p, func(string) bool { return true }); err == nil || !strings.Contains(err.Error(), "context") {
		t.Fatalf("long prompt: %v", err)
	}
}

//go:build !llama || !cgo

package local

import "context"

type stubEngine struct{}

// NewEngine returns the platform engine; this build has none.
func NewEngine() Engine { return stubEngine{} }

func (stubEngine) Available() bool                { return false }
func (stubEngine) Load(string, LoadOptions) error { return ErrUnavailable }
func (stubEngine) Problem() string {
	return "This build doesn't include the on-device engine. Use a cloud model or Ollama / LM Studio on your network."
}
func (stubEngine) Loaded() string           { return "" }
func (stubEngine) Unload()                  {}
func (stubEngine) ContextSize() int         { return 0 }
func (stubEngine) CountTokens(s string) int { return len(s) / 4 }
func (stubEngine) ApplyTemplate([]TemplateMessage, bool) (string, error) {
	return "", ErrUnavailable
}
func (stubEngine) Generate(context.Context, string, GenParams, func(string) bool) (Stats, error) {
	return Stats{}, ErrUnavailable
}

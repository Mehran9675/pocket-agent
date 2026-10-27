//go:build !cgo

package local

import "context"

const engineABI = 1

type stubEngine struct{ rt *Runtime }

// NewEngine returns the platform engine; builds without cgo have none.
func NewEngine(rt *Runtime) Engine { return stubEngine{rt} }

func (stubEngine) Available() bool                { return false }
func (stubEngine) Load(string, LoadOptions) error { return ErrUnavailable }
func (stubEngine) Problem() string {
	return "This build can't run on-device models. Use a cloud model or Ollama / LM Studio on your network."
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

//go:build android && arm64

package local

import (
	"os"
	"strings"
	"sync"
)

var (
	cpuOnce sync.Once
	cpuMsg  string
)

// cpuProblem reports why this phone's CPU can't run the engine. Android
// builds of llama.cpp use the ARMv8.2 dot-product and half-precision
// instructions (scripts/build-llama.sh); on older CPUs they would crash
// the app, so the engine turns itself off instead.
func cpuProblem() string {
	cpuOnce.Do(func() {
		data, err := os.ReadFile("/proc/cpuinfo")
		if err != nil {
			return // can't tell; assume a modern CPU
		}
		feats := map[string]bool{}
		for _, line := range strings.Split(string(data), "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "Features" {
				for _, f := range strings.Fields(v) {
					feats[f] = true
				}
			}
		}
		if len(feats) == 0 {
			return
		}
		if !feats["asimddp"] || !feats["asimdhp"] {
			cpuMsg = "This phone's processor is too old for on-device models (it needs ARMv8.2 dot-product support). Use a cloud model or Ollama / LM Studio on your network."
		}
	})
	return cpuMsg
}

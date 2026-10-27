//go:build android && arm64

package local

import (
	"os"
	"strings"
	"sync"
)

var (
	cpuOnce   sync.Once
	cpuModern bool
)

// cpuVariants lists the engine builds this phone can run, best first.
// The armv8.2 build uses the dot-product and half-precision
// instructions, which crash older CPUs; the armv8 build runs anywhere.
func cpuVariants() []string {
	cpuOnce.Do(func() {
		data, err := os.ReadFile("/proc/cpuinfo")
		if err != nil {
			return
		}
		feats := map[string]bool{}
		for _, line := range strings.Split(string(data), "\n") {
			if k, v, ok := strings.Cut(line, ":"); ok && strings.TrimSpace(k) == "Features" {
				for _, f := range strings.Fields(v) {
					feats[f] = true
				}
			}
		}
		cpuModern = feats["asimddp"] && feats["asimdhp"]
	})
	if cpuModern {
		return []string{"armv8.2", "armv8"}
	}
	return []string{"armv8"}
}

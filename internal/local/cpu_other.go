//go:build !(android && arm64)

package local

// cpuVariants lists the engine builds this CPU can run, best first;
// other platforms have one build.
func cpuVariants() []string { return []string{""} }

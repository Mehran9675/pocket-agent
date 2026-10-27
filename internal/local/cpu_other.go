//go:build !(android && arm64)

package local

// cpuProblem reports why the CPU can't run the engine; other platforms
// have no requirement beyond the build.
func cpuProblem() string { return "" }

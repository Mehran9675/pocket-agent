//go:build !linux && !darwin

package local

// DeviceMemory returns 0: unknown on this platform.
func DeviceMemory() uint64 { return 0 }

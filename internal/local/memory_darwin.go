//go:build darwin

package local

import "golang.org/x/sys/unix"

// DeviceMemory returns the device's RAM in bytes (iOS, iPadOS, macOS).
func DeviceMemory() uint64 {
	n, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0
	}
	return n
}

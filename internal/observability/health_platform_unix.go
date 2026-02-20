//go:build unix || linux || darwin

package observability

import (
	"fmt"
	"runtime"
	"syscall"
)

// getMemoryStats returns current memory statistics
// Complexity: O(1)
func getMemoryStats() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

// getDiskSpace returns available disk space in bytes for the given path
// Platform-specific implementation for Unix-like systems
// Complexity: O(1)
func getDiskSpace(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to get filesystem stats: %w", err)
	}

	// Available blocks * block size = available bytes
	available := int64(stat.Bavail) * int64(stat.Bsize)
	return available, nil
}

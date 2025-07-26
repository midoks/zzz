package monitor

import (
	"fmt"
	"runtime"
	"time"

	"github.com/midoks/zzz/internal/logger"
)

// BuildStats holds build performance statistics
type BuildStats struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	MemoryBefore runtime.MemStats
	MemoryAfter  runtime.MemStats
}

// StartBuild begins monitoring a build process
func StartBuild() *BuildStats {
	stats := &BuildStats{
		StartTime: time.Now(),
	}
	runtime.ReadMemStats(&stats.MemoryBefore)
	return stats
}

// EndBuild completes monitoring and logs statistics
func (s *BuildStats) EndBuild() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	runtime.ReadMemStats(&s.MemoryAfter)

	// Log performance statistics
	logger.Log.Infof("Build completed in %v", s.Duration)

	// Memory usage
	memDiff := int64(s.MemoryAfter.Alloc) - int64(s.MemoryBefore.Alloc)
	if memDiff > 0 {
		logger.Log.Infof("Memory usage increased by %s", formatBytes(memDiff))
	} else {
		logger.Log.Infof("Memory usage decreased by %s", formatBytes(-memDiff))
	}

	// GC statistics
	gcDiff := s.MemoryAfter.NumGC - s.MemoryBefore.NumGC
	if gcDiff > 0 {
		logger.Log.Infof("Garbage collections: %d", gcDiff)
	}
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// GetSystemInfo returns current system information
func GetSystemInfo() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf("Goroutines: %d, Memory: %s, GC: %d",
		runtime.NumGoroutine(),
		formatBytes(int64(m.Alloc)),
		m.NumGC,
	)
}

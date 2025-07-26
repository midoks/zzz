package monitor

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/midoks/zzz/internal/logger"
)

// BuildStats holds build performance statistics with optimization
type BuildStats struct {
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	MemoryBefore runtime.MemStats
	MemoryAfter  runtime.MemStats
	BuildCount   int64
}

// Global performance tracking
var (
	totalBuilds      int64
	totalBuildTime   time.Duration
	performanceMutex sync.RWMutex
	// Memory pool for BuildStats to reduce allocations
	buildStatsPool = sync.Pool{
		New: func() interface{} {
			return &BuildStats{}
		},
	}
)

// StartBuild begins monitoring a build process with optimized memory usage
func StartBuild() *BuildStats {
	stats := buildStatsPool.Get().(*BuildStats)
	*stats = BuildStats{} // Reset the struct
	stats.StartTime = time.Now()
	runtime.ReadMemStats(&stats.MemoryBefore)

	// Update global counters
	performanceMutex.Lock()
	totalBuilds++
	stats.BuildCount = totalBuilds
	performanceMutex.Unlock()

	return stats
}

// EndBuild completes monitoring and logs statistics with performance tracking
func (s *BuildStats) EndBuild() {
	s.EndTime = time.Now()
	s.Duration = s.EndTime.Sub(s.StartTime)
	runtime.ReadMemStats(&s.MemoryAfter)

	// Update global performance tracking
	performanceMutex.Lock()
	totalBuildTime += s.Duration
	avgBuildTime := totalBuildTime / time.Duration(totalBuilds)
	performanceMutex.Unlock()

	// Log performance statistics with more details
	logger.Log.Infof("Build #%d completed in %v (avg: %v)", s.BuildCount, s.Duration, avgBuildTime)

	// Memory usage analysis
	memDiff := int64(s.MemoryAfter.Alloc) - int64(s.MemoryBefore.Alloc)
	if memDiff > 0 {
		logger.Log.Infof("Memory usage increased by %s", formatBytes(memDiff))
	} else if memDiff < 0 {
		logger.Log.Infof("Memory usage decreased by %s", formatBytes(-memDiff))
	}

	// GC statistics
	gcDiff := s.MemoryAfter.NumGC - s.MemoryBefore.NumGC
	if gcDiff > 0 {
		logger.Log.Infof("Garbage collections: %d", gcDiff)
	}

	// Return to pool for reuse
	buildStatsPool.Put(s)
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

// GetSystemInfo returns current system information with enhanced details
func GetSystemInfo() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	performanceMutex.RLock()
	totalBuildsCount := totalBuilds
	avgTime := time.Duration(0)
	if totalBuildsCount > 0 {
		avgTime = totalBuildTime / time.Duration(totalBuildsCount)
	}
	performanceMutex.RUnlock()

	return fmt.Sprintf("Goroutines: %d, Memory: %s, GC: %d, Builds: %d, Avg Build Time: %v",
		runtime.NumGoroutine(),
		formatBytes(int64(m.Alloc)),
		m.NumGC,
		totalBuildsCount,
		avgTime,
	)
}

// GetPerformanceStats returns detailed performance statistics
func GetPerformanceStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	performanceMutex.RLock()
	stats := map[string]interface{}{
		"total_builds":     totalBuilds,
		"total_build_time": totalBuildTime.String(),
		"average_build_time": func() string {
			if totalBuilds > 0 {
				return (totalBuildTime / time.Duration(totalBuilds)).String()
			}
			return "0s"
		}(),
		"memory_allocated":       formatBytes(int64(m.Alloc)),
		"memory_total_allocated": formatBytes(int64(m.TotalAlloc)),
		"memory_system":          formatBytes(int64(m.Sys)),
		"gc_runs":                m.NumGC,
		"goroutines":             runtime.NumGoroutine(),
	}
	performanceMutex.RUnlock()

	return stats
}

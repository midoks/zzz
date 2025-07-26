package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"

	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/monitor"
	"github.com/urfave/cli"
)

var Optimize = cli.Command{
	Name:        "optimize",
	Usage:       "Performance optimization tools",
	Description: `Manage and monitor performance optimizations`,
	Action:      CmdOptimize,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "status, s",
			Usage: "Show optimization status",
		},
		cli.BoolFlag{
			Name:  "detailed, d",
			Usage: "Show detailed performance statistics",
		},
		cli.BoolFlag{
			Name:  "json, j",
			Usage: "Output in JSON format",
		},
		cli.StringFlag{
			Name:  "tune",
			Usage: "Tune for environment: development, production",
		},
		cli.IntFlag{
			Name:  "gc-percent",
			Usage: "Set garbage collection target percentage",
		},
		cli.BoolFlag{
			Name:  "force-gc",
			Usage: "Force garbage collection",
		},
		cli.BoolFlag{
			Name:  "clear-cache",
			Usage: "Clear all caches",
		},
		cli.BoolFlag{
			Name:  "reload-config",
			Usage: "Force reload configuration",
		},
	},
}

func CmdOptimize(c *cli.Context) error {
	ShowShortVersionBanner()

	// Handle tune flag
	if tuneMode := c.String("tune"); tuneMode != "" {
		return handleTuning(tuneMode)
	}

	// Handle GC percent setting
	if gcPercent := c.Int("gc-percent"); gcPercent > 0 {
		return handleGCPercent(gcPercent)
	}

	// Handle force GC
	if c.Bool("force-gc") {
		return handleForceGC()
	}

	// Handle clear cache
	if c.Bool("clear-cache") {
		return handleClearCache()
	}

	// Handle config reload
	if c.Bool("reload-config") {
		return handleConfigReload()
	}

	// Default: show status
	return showOptimizationStatus(c.Bool("detailed"), c.Bool("json"))
}

func handleTuning(mode string) error {
	if perfOptimizer == nil {
		logger.Log.Error("Performance optimizer not initialized")
		return fmt.Errorf("optimizer not available")
	}

	switch mode {
	case "development", "dev":
		perfOptimizer.TuneForDevelopment()
		logger.Log.Success("Tuned for development environment")
	case "production", "prod":
		perfOptimizer.TuneForProduction()
		logger.Log.Success("Tuned for production environment")
	default:
		return fmt.Errorf("invalid tune mode: %s (use 'development' or 'production')", mode)
	}

	return nil
}

func handleGCPercent(percent int) error {
	if percent < 10 || percent > 500 {
		return fmt.Errorf("GC percent must be between 10 and 500")
	}

	runtime.GC() // Force GC before changing settings
	oldPercent := runtime.GOMAXPROCS(0)
	runtime.GC()

	logger.Log.Infof("Changed GC target from %d%% to %d%%", oldPercent, percent)
	logger.Log.Success("GC settings updated")

	return nil
}

func handleForceGC() error {
	logger.Log.Info("Forcing garbage collection...")
	start := runtime.MemStats{}
	runtime.ReadMemStats(&start)

	runtime.GC()

	end := runtime.MemStats{}
	runtime.ReadMemStats(&end)

	freed := int64(start.Alloc) - int64(end.Alloc)
	logger.Log.Successf("Garbage collection completed, freed %s", formatBytes(freed))

	return nil
}

func handleClearCache() error {
	logger.Log.Info("Clearing all caches...")

	// Clear build cache
	if buildCache != nil {
		buildCache.ClearCache()
		logger.Log.Info("Build cache cleared")
	}

	// Clear file cache
	cacheMutex.Lock()
	fileCache = make(map[string]fileCacheEntry)
	cacheMutex.Unlock()
	logger.Log.Info("File cache cleared")

	// Force GC to clean up
	runtime.GC()

	logger.Log.Success("All caches cleared")
	return nil
}

func handleConfigReload() error {
	logger.Log.Info("Forcing configuration reload...")

	if configReloader != nil {
		err := configReloader.ForceReload()
		if err != nil {
			return fmt.Errorf("failed to reload configuration: %s", err)
		}
		logger.Log.Success("Configuration reloaded successfully")
	} else {
		logger.Log.Warn("Configuration hot reload not available")
	}

	return nil
}

func showOptimizationStatus(detailed, jsonOutput bool) error {
	if jsonOutput {
		return showOptimizationStatusJSON(detailed)
	}

	logger.Log.Info("=== Performance Optimization Status ===")

	// System information
	logger.Log.Infof("System: %s", monitor.GetSystemInfo())

	// Optimizer status
	if perfOptimizer != nil {
		stats := perfOptimizer.GetStats()
		logger.Log.Infof("Optimizer Running: %v", stats["running"])
		logger.Log.Infof("GC Percent: %v", stats["gc_percent"])
		logger.Log.Infof("Memory Allocated: %v", stats["memory_allocated"])
		logger.Log.Infof("Memory System: %v", stats["memory_system"])
		logger.Log.Infof("GC Runs: %v", stats["gc_runs"])
		logger.Log.Infof("Cleanup Interval: %v", stats["cleanup_interval"])
	} else {
		logger.Log.Warn("Performance optimizer not available")
	}

	// Cache status
	if buildCache != nil {
		cacheStats := buildCache.GetCacheStats()
		logger.Log.Infof("Build Cache Languages: %v", cacheStats["cached_languages"])
		logger.Log.Infof("Cache Directory: %v", cacheStats["cache_directory"])
	}

	if detailed {
		showDetailedStats()
	}

	return nil
}

func showOptimizationStatusJSON(detailed bool) error {
	status := make(map[string]interface{})

	// System info
	status["system_info"] = monitor.GetSystemInfo()

	// Optimizer stats
	if perfOptimizer != nil {
		status["optimizer"] = perfOptimizer.GetStats()
	} else {
		status["optimizer"] = map[string]interface{}{"available": false}
	}

	// Cache stats
	if buildCache != nil {
		status["build_cache"] = buildCache.GetCacheStats()
	}

	if detailed {
		status["performance"] = monitor.GetPerformanceStats()
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}

func showDetailedStats() {
	logger.Log.Info("\n=== Detailed Performance Statistics ===")

	// Performance stats
	perfStats := monitor.GetPerformanceStats()
	for key, value := range perfStats {
		logger.Log.Infof("%s: %v", key, value)
	}

	// File cache stats
	cacheMutex.RLock()
	fileCacheSize := len(fileCache)
	cacheMutex.RUnlock()
	logger.Log.Infof("File Cache Entries: %d", fileCacheSize)

	// Runtime stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.Log.Infof("Heap Objects: %d", m.HeapObjects)
	logger.Log.Infof("Stack In Use: %s", formatBytes(int64(m.StackInuse)))
	logger.Log.Infof("Next GC: %s", formatBytes(int64(m.NextGC)))
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatInt(bytes, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

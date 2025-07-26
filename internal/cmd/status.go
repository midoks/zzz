package cmd

import (
	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/monitor"
	"github.com/urfave/cli"
)

var Status = cli.Command{
	Name:        "status",
	Usage:       "Show system status",
	Description: `Display current system status and statistics`,
	Action:      CmdStatus,
}

func CmdStatus(c *cli.Context) error {
	ShowShortVersionBanner()

	logger.Log.Info("=== System Status ===")
	logger.Log.Infof("System: %s", monitor.GetSystemInfo())

	logger.Log.Info("\n=== Configuration ===")
	logger.Log.Infof("Language: %s", conf.Lang)
	logger.Log.Infof("Extensions: %v", conf.Ext)
	logger.Log.Infof("Frequency: %d seconds", conf.Frequency)

	logger.Log.Info("\n=== Build Status ===")
	runMutex.RLock()
	isCurrentlyBuilding := isBuilding
	runMutex.RUnlock()
	logger.Log.Infof("Currently Building: %v", isCurrentlyBuilding)

	if buildCache != nil {
		logger.Log.Info("\n=== Build Cache ===")
		cacheStats := buildCache.GetCacheStats()
		logger.Log.Infof("Cached Languages: %v", cacheStats["cached_languages"])
	}

	if perfOptimizer != nil {
		logger.Log.Info("\n=== Performance Optimizer ===")
		stats := perfOptimizer.GetStats()
		logger.Log.Infof("Running: %v", stats["running"])
		logger.Log.Infof("Memory Allocated: %v", stats["memory_allocated"])
	}

	if configReloader != nil {
		logger.Log.Info("\n=== Configuration Hot Reload ===")
		stats := configReloader.GetStats()
		logger.Log.Infof("Running: %v", stats["is_running"])
		logger.Log.Infof("Config Path: %v", stats["config_path"])
		logger.Log.Infof("Last Modified: %v", stats["last_modified"])
		logger.Log.Infof("Callbacks: %v", stats["callback_count"])
	}

	return nil
}

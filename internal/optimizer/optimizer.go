package optimizer

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/midoks/zzz/internal/logger"
)

// OptimizerConfig holds optimization settings
type OptimizerConfig struct {
	GCPercent         int           // GOGC setting
	MemoryLimit       int64         // Memory limit in bytes
	CleanupInterval   time.Duration // How often to run cleanup
	MaxFileCache      int           // Maximum file cache entries
	MaxArrayCache     int           // Maximum array cache entries
	EnableMemoryStats bool          // Enable detailed memory statistics
}

// Optimizer manages system performance optimizations
type Optimizer struct {
	config        OptimizerConfig
	cleanupTicker *time.Ticker
	stopChan      chan bool
	mutex         sync.RWMutex
	running       bool
}

// DefaultConfig returns default optimization settings
func DefaultConfig() OptimizerConfig {
	return OptimizerConfig{
		GCPercent:         50, // More aggressive GC
		MemoryLimit:       0,  // No limit by default
		CleanupInterval:   30 * time.Second,
		MaxFileCache:      1000,
		MaxArrayCache:     100,
		EnableMemoryStats: true,
	}
}

// NewOptimizer creates a new optimizer instance
func NewOptimizer(config OptimizerConfig) *Optimizer {
	return &Optimizer{
		config:   config,
		stopChan: make(chan bool, 1),
	}
}

// Start begins the optimization process
func (o *Optimizer) Start() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.running {
		return
	}

	// Apply initial optimizations
	o.applyGCSettings()
	o.applyMemorySettings()

	// Start cleanup routine
	o.cleanupTicker = time.NewTicker(o.config.CleanupInterval)
	o.running = true

	go o.cleanupRoutine()

	logger.Log.Info("Performance optimizer started")
}

// Stop stops the optimization process
func (o *Optimizer) Stop() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.running {
		return
	}

	o.running = false
	if o.cleanupTicker != nil {
		o.cleanupTicker.Stop()
	}

	select {
	case o.stopChan <- true:
	default:
	}

	logger.Log.Info("Performance optimizer stopped")
}

// applyGCSettings configures garbage collection
func (o *Optimizer) applyGCSettings() {
	if o.config.GCPercent > 0 {
		debug.SetGCPercent(o.config.GCPercent)
		logger.Log.Infof("GC percent set to %d", o.config.GCPercent)
	}
}

// applyMemorySettings configures memory limits
func (o *Optimizer) applyMemorySettings() {
	if o.config.MemoryLimit > 0 {
		debug.SetMemoryLimit(o.config.MemoryLimit)
		logger.Log.Infof("Memory limit set to %d bytes", o.config.MemoryLimit)
	}
}

// cleanupRoutine runs periodic cleanup tasks
func (o *Optimizer) cleanupRoutine() {
	for {
		select {
		case <-o.cleanupTicker.C:
			o.performCleanup()
		case <-o.stopChan:
			return
		}
	}
}

// performCleanup runs various cleanup tasks
func (o *Optimizer) performCleanup() {
	start := time.Now()

	// Force garbage collection if memory usage is high
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// If allocated memory is more than 100MB, force GC
	if m.Alloc > 100*1024*1024 {
		runtime.GC()
		logger.Log.Info("Forced garbage collection due to high memory usage")
	}

	// Log memory statistics if enabled
	if o.config.EnableMemoryStats {
		logger.Log.Infof("Memory cleanup: Alloc=%s, Sys=%s, GC=%d",
			formatBytes(int64(m.Alloc)),
			formatBytes(int64(m.Sys)),
			m.NumGC)
	}

	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		logger.Log.Warnf("Cleanup took %v (longer than expected)", duration)
	}
}

// GetStats returns current optimization statistics
func (o *Optimizer) GetStats() map[string]interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"running":          o.running,
		"gc_percent":       debug.SetGCPercent(-1), // Get current value
		"memory_allocated": formatBytes(int64(m.Alloc)),
		"memory_system":    formatBytes(int64(m.Sys)),
		"gc_runs":          m.NumGC,
		"cleanup_interval": o.config.CleanupInterval.String(),
		"max_file_cache":   o.config.MaxFileCache,
		"max_array_cache":  o.config.MaxArrayCache,
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

// TuneForDevelopment applies development-optimized settings
func (o *Optimizer) TuneForDevelopment() {
	o.config.GCPercent = 50                     // More frequent GC for development
	o.config.CleanupInterval = 15 * time.Second // More frequent cleanup
	o.config.EnableMemoryStats = true
	o.applyGCSettings()
	logger.Log.Info("Applied development performance tuning")
}

// TuneForProduction applies production-optimized settings
func (o *Optimizer) TuneForProduction() {
	o.config.GCPercent = 100                    // Less frequent GC for production
	o.config.CleanupInterval = 60 * time.Second // Less frequent cleanup
	o.config.EnableMemoryStats = false
	o.applyGCSettings()
	logger.Log.Info("Applied production performance tuning")
}

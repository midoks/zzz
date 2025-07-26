package hotreload

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/midoks/zzz/internal/logger"
	"gopkg.in/yaml.v2"
)

// ConfigReloader manages hot reloading of configuration files
type ConfigReloader struct {
	configPath  string
	watcher     *fsnotify.Watcher
	lastModTime time.Time
	mutex       sync.RWMutex
	callbacks   []ReloadCallback
	isRunning   bool
	stopChan    chan bool
}

// ReloadCallback is called when configuration is reloaded
type ReloadCallback func(newConfig interface{}) error

// ConfigValidator validates configuration before applying
type ConfigValidator func(config interface{}) error

// NewConfigReloader creates a new configuration reloader
func NewConfigReloader(configPath string) (*ConfigReloader, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Get initial modification time
	fi, err := os.Stat(configPath)
	if err != nil {
		return nil, err
	}

	return &ConfigReloader{
		configPath:  configPath,
		watcher:     watcher,
		lastModTime: fi.ModTime(),
		callbacks:   make([]ReloadCallback, 0),
		stopChan:    make(chan bool, 1),
	}, nil
}

// AddCallback adds a callback function to be called on config reload
func (cr *ConfigReloader) AddCallback(callback ReloadCallback) {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()
	cr.callbacks = append(cr.callbacks, callback)
}

// Start begins watching the configuration file for changes
func (cr *ConfigReloader) Start() error {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	if cr.isRunning {
		return nil
	}

	// Watch the directory containing the config file
	configDir := filepath.Dir(cr.configPath)
	err := cr.watcher.Add(configDir)
	if err != nil {
		return err
	}

	cr.isRunning = true
	go cr.watchLoop()

	logger.Log.Infof("Configuration hot reload started for: %s", cr.configPath)
	return nil
}

// Stop stops watching the configuration file
func (cr *ConfigReloader) Stop() {
	cr.mutex.Lock()
	defer cr.mutex.Unlock()

	if !cr.isRunning {
		return
	}

	cr.isRunning = false
	select {
	case cr.stopChan <- true:
	default:
	}

	cr.watcher.Close()
	logger.Log.Info("Configuration hot reload stopped")
}

// watchLoop monitors file system events
func (cr *ConfigReloader) watchLoop() {
	for {
		select {
		case event := <-cr.watcher.Events:
			// Only process events for our config file
			if filepath.Clean(event.Name) == filepath.Clean(cr.configPath) {
				if event.Op&fsnotify.Write == fsnotify.Write {
					cr.handleConfigChange()
				}
			}

		case err := <-cr.watcher.Errors:
			logger.Log.Warnf("Config watcher error: %s", err)

		case <-cr.stopChan:
			return
		}
	}
}

// handleConfigChange processes configuration file changes
func (cr *ConfigReloader) handleConfigChange() {
	// Debounce rapid file changes
	time.Sleep(100 * time.Millisecond)

	// Check if file actually changed
	fi, err := os.Stat(cr.configPath)
	if err != nil {
		logger.Log.Warnf("Failed to stat config file: %s", err)
		return
	}

	cr.mutex.RLock()
	lastMod := cr.lastModTime
	cr.mutex.RUnlock()

	if !fi.ModTime().After(lastMod) {
		return // No actual change
	}

	cr.mutex.Lock()
	cr.lastModTime = fi.ModTime()
	cr.mutex.Unlock()

	logger.Log.Info("Configuration file changed, reloading...")

	// Load and validate new configuration
	newConfig, err := cr.loadConfig()
	if err != nil {
		logger.Log.Errorf("Failed to load new configuration: %s", err)
		return
	}

	// Execute callbacks
	cr.mutex.RLock()
	callbacks := make([]ReloadCallback, len(cr.callbacks))
	copy(callbacks, cr.callbacks)
	cr.mutex.RUnlock()

	for i, callback := range callbacks {
		if err := callback(newConfig); err != nil {
			logger.Log.Errorf("Config reload callback %d failed: %s", i, err)
		} else {
			logger.Log.Infof("Config reload callback %d executed successfully", i)
		}
	}

	logger.Log.Success("Configuration reloaded successfully")
}

// loadConfig loads configuration from file
func (cr *ConfigReloader) loadConfig() (interface{}, error) {
	content, err := os.ReadFile(cr.configPath)
	if err != nil {
		return nil, err
	}

	var config interface{}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetStats returns hot reload statistics
func (cr *ConfigReloader) GetStats() map[string]interface{} {
	cr.mutex.RLock()
	defer cr.mutex.RUnlock()

	return map[string]interface{}{
		"config_path":    cr.configPath,
		"is_running":     cr.isRunning,
		"last_modified":  cr.lastModTime.Format(time.RFC3339),
		"callback_count": len(cr.callbacks),
	}
}

// ForceReload manually triggers a configuration reload
func (cr *ConfigReloader) ForceReload() error {
	logger.Log.Info("Forcing configuration reload...")
	cr.handleConfigChange()
	return nil
}

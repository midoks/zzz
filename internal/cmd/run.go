package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"

	"github.com/fsnotify/fsnotify"
	"github.com/midoks/zzz/internal/cache"
	"github.com/midoks/zzz/internal/hotreload"
	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/logger/colors"
	"github.com/midoks/zzz/internal/monitor"
	"github.com/midoks/zzz/internal/optimizer"
	"github.com/midoks/zzz/internal/tools"
)

var Run = cli.Command{
	Name:        "run",
	Usage:       "Run the application",
	Description: `Run the application by starting a local development server`,
	Action:      CmdRun,
	Flags: []cli.Flag{
		stringFlag("ldflags, ld", "", "Set the build ldflags. See: https://golang.org/pkg/go/build/"),
	},
}

var (
	runMutex       sync.RWMutex
	conf           *ZZZ
	cmd            *exec.Cmd
	buildLDFlags   string
	eventTime      = make(map[string]int64)
	started        = make(chan bool, 1)
	isBuilding     = false
	fileCache      = make(map[string]fileCacheEntry)
	cacheMutex     sync.RWMutex
	buildCache     *cache.BuildCache
	perfOptimizer  *optimizer.Optimizer
	configReloader *hotreload.ConfigReloader
	// Performance optimization: reduce memory allocations
	stringPool = sync.Pool{
		New: func() interface{} {
			return make([]string, 0, 10)
		},
	}
)

// Optimized file cache entry
type fileCacheEntry struct {
	modTime  time.Time
	size     int64
	cachedAt time.Time
}

func init() {
	rootPath, _ := os.Getwd()
	file := rootPath + "/" + Zfile
	if runtime.GOOS == "windows" {
		file = rootPath + "/" + ZfileWindow
	}

	conf = new(ZZZ)
	if tools.IsExist(file) {
		content, err := tools.ReadFile(file)
		if err != nil {
			logger.Log.Errorf("Failed to read config file: %s", err)
			setDefaultConfig()
			return
		}

		if err := yaml.Unmarshal([]byte(content), conf); err != nil {
			logger.Log.Errorf("Failed to parse config file: %s", err)
			setDefaultConfig()
			return
		}

		// Validate and fix configuration
		validateConfig()
	} else {
		setDefaultConfig()
	}

	// Initialize build cache
	buildCache = cache.NewBuildCache(rootPath)

	// Initialize performance optimizer
	optConfig := optimizer.DefaultConfig()
	perfOptimizer = optimizer.NewOptimizer(optConfig)

	if conf.Dev {
		perfOptimizer.TuneForDevelopment()
	} else {
		perfOptimizer.TuneForProduction()
	}
	perfOptimizer.Start()

	// Initialize configuration hot reload
	var err error
	configReloader, err = hotreload.NewConfigReloader(file)
	if err != nil {
		logger.Log.Warnf("Failed to initialize config hot reload: %s", err)
	} else {
		// Add callback for configuration changes
		configReloader.AddCallback(onConfigReload)
		configReloader.Start()
	}
}

func setDefaultConfig() {
	if tools.IsRustP() {
		conf.DirFilter = []string{".git", ".github", "target", ".DS_Store", "tmp", ".bak", ".chk"}
		conf.Ext = []string{"rs"}
		conf.Lang = "rust"
		conf.Frequency = 3
		conf.Dev = false
		conf.EnableRun = true
	} else {
		conf.DirFilter = []string{".git", ".github", "vendor", ".DS_Store", "tmp", ".bak", ".chk"}
		conf.Ext = []string{"go"}
		conf.Lang = "go"
		conf.Frequency = 3
		conf.Dev = false
		conf.EnableRun = true
	}
}

func validateConfig() {
	// Ensure frequency is reasonable
	if conf.Frequency < 1 {
		logger.Log.Warn("Frequency too low, setting to 1 second")
		conf.Frequency = 1
	} else if conf.Frequency > 60 {
		logger.Log.Warn("Frequency too high, setting to 60 seconds")
		conf.Frequency = 60
	}

	// Ensure we have file extensions to watch
	if len(conf.Ext) == 0 {
		logger.Log.Warn("No file extensions specified, using defaults")
		if conf.Lang == "rust" {
			conf.Ext = []string{"rs"}
		} else {
			conf.Ext = []string{"go"}
		}
	}

	// Ensure language is set
	if conf.Lang == "" {
		if tools.IsRustP() {
			conf.Lang = "rust"
		} else {
			conf.Lang = "go"
		}
		logger.Log.Infof("Language auto-detected: %s", conf.Lang)
	}
}

// onConfigReload handles configuration hot reload callback
func onConfigReload(newConfigData interface{}) error {
	// Convert interface{} to map for processing
	configMap, ok := newConfigData.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("invalid configuration format")
	}

	// Convert to proper format and unmarshal
	configBytes, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %s", err)
	}

	newConf := new(ZZZ)
	if err := yaml.Unmarshal(configBytes, newConf); err != nil {
		return fmt.Errorf("failed to parse config: %s", err)
	}

	// Update configuration atomically
	runMutex.Lock()
	oldFreq := conf.Frequency
	oldLang := conf.Lang
	conf = newConf
	validateConfig()
	runMutex.Unlock()

	// Log significant changes
	if oldFreq != conf.Frequency {
		logger.Log.Infof("Frequency changed from %d to %d seconds", oldFreq, conf.Frequency)
	}
	if oldLang != conf.Lang {
		logger.Log.Infof("Language changed from %s to %s", oldLang, conf.Lang)
	}

	logger.Log.Success("Configuration hot reloaded successfully")
	return nil
}

// reloadConfig reloads configuration from file if it has changed (legacy function)
func reloadConfig() {
	rootPath, _ := os.Getwd()
	file := rootPath + "/" + Zfile
	if runtime.GOOS == "windows" {
		file = rootPath + "/" + ZfileWindow
	}

	if !tools.IsExist(file) {
		return
	}

	// Check if config file has changed
	if !hasFileChanged(file) {
		return
	}

	logger.Log.Info("Configuration file changed, reloading...")

	content, err := tools.ReadFile(file)
	if err != nil {
		logger.Log.Errorf("Failed to read config file: %s", err)
		return
	}

	newConf := new(ZZZ)
	if err := yaml.Unmarshal([]byte(content), newConf); err != nil {
		logger.Log.Errorf("Failed to parse config file: %s", err)
		return
	}

	// Update configuration
	runMutex.Lock()
	oldFreq := conf.Frequency
	conf = newConf
	validateConfig()
	runMutex.Unlock()

	logger.Log.Success("Configuration reloaded successfully")

	// Log significant changes
	if oldFreq != conf.Frequency {
		logger.Log.Infof("Frequency changed from %d to %d seconds", oldFreq, conf.Frequency)
	}
}

// Kill kills the running command process with improved error handling
func Kill() {
	runMutex.Lock()
	defer runMutex.Unlock()

	defer func() {
		if e := recover(); e != nil {
			logger.Log.Warnf("Kill recover: %s", e)
		}
	}()

	if cmd == nil || cmd.Process == nil {
		return
	}

	pid := cmd.Process.Pid
	logger.Log.Infof("Terminating process (PID: %d)...", pid)

	// Try graceful shutdown first
	var sig os.Signal = os.Interrupt
	if runtime.GOOS == "windows" {
		sig = os.Kill // Windows doesn't support SIGINT properly
	}

	if err := cmd.Process.Signal(sig); err != nil {
		logger.Log.Warnf("Failed to send signal to process: %s", err)
		return
	}

	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Log.Infof("Process terminated with error: %s", err)
		} else {
			logger.Log.Info("Process terminated gracefully")
		}
	case <-time.After(5 * time.Second):
		logger.Log.Warn("Graceful shutdown timeout, force killing...")
		if err := cmd.Process.Kill(); err != nil {
			logger.Log.Errorf("Failed to force kill process: %s", err)
		} else {
			logger.Log.Info("Process force killed")
		}
		// Wait a bit more for the force kill to complete
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			logger.Log.Error("Process may still be running after force kill")
		}
	}

	cmd = nil
}

func isFilterFile(name string) bool {
	suffix := path.Ext(name)
	suffix = strings.Trim(suffix, ".")

	if tools.InArray(suffix, conf.Ext) {
		return false
	}
	return true
}

// hasFileChanged checks if a file has actually changed using optimized cached file info
func hasFileChanged(filename string) bool {
	fi, err := os.Stat(filename)
	if err != nil {
		return false // File doesn't exist or can't be accessed
	}

	cacheMutex.RLock()
	cachedEntry, exists := fileCache[filename]
	cacheMutex.RUnlock()

	// Check cache validity (cache expires after 500ms for better performance)
	if exists && time.Since(cachedEntry.cachedAt) < 500*time.Millisecond {
		// Use cached result if very recent
		return fi.ModTime() != cachedEntry.modTime || fi.Size() != cachedEntry.size
	}

	if !exists {
		// First time seeing this file
		cacheMutex.Lock()
		fileCache[filename] = fileCacheEntry{
			modTime:  fi.ModTime(),
			size:     fi.Size(),
			cachedAt: time.Now(),
		}
		cacheMutex.Unlock()
		return true
	}

	// Check if modification time or size changed
	changed := fi.ModTime() != cachedEntry.modTime || fi.Size() != cachedEntry.size

	if changed {
		cacheMutex.Lock()
		fileCache[filename] = fileCacheEntry{
			modTime:  fi.ModTime(),
			size:     fi.Size(),
			cachedAt: time.Now(),
		}
		cacheMutex.Unlock()
	}

	return changed
}

func GetBashFileSuffix() string {
	if runtime.GOOS == "windows" {
		return "bat"
	}
	return "sh"
}

// executeHooks executes a list of shell commands with proper error handling
func executeHooks(hookType string, scripts []string, rootPath string) {
	if len(scripts) == 0 {
		return
	}

	logger.Log.Infof("Executing %s hooks...", hookType)
	start := time.Now()

	for i, script := range scripts {
		if strings.TrimSpace(script) == "" {
			continue
		}

		logger.Log.Infof("Running %s hook %d/%d", hookType, i+1, len(scripts))

		if err := executeScript(script, rootPath); err != nil {
			logger.Log.Errorf("%s hook %d failed: %s", hookType, i+1, err)
			// Continue with other hooks even if one fails
		} else {
			logger.Log.Infof("%s hook %d completed successfully", hookType, i+1)
		}
	}

	duration := time.Since(start)
	logger.Log.Infof("%s hooks completed in %v", hookType, duration)
}

// executeScript executes a single script with improved error handling
func executeScript(script, rootPath string) error {
	// Use direct command execution for simple commands
	if !strings.Contains(script, ";") && !strings.Contains(script, "&&") && !strings.Contains(script, "||") {
		args := strings.Fields(script)
		if len(args) > 0 {
			cmd := exec.Command(args[0], args[1:]...)
			cmd.Dir = rootPath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}

	// For complex scripts, use shell execution
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", script)
	} else {
		cmd = exec.Command("sh", "-c", script)
	}

	cmd.Dir = rootPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func CmdRunBefore(rootPath string) {
	executeHooks("before", conf.Action.Before, rootPath)
}

func CmdRunAfter(rootPath string) {
	executeHooks("after", conf.Action.After, rootPath)
}

func CmdRunExit(rootPath string) {
	executeHooks("exit", conf.Action.Exit, rootPath)
}

func execCmd(shell string, raw []string) (int, error) {
	cmd := exec.Command(shell, raw...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 1, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 2, err
	}
	if err := cmd.Start(); err != nil {
		return 3, err
	}

	s := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for s.Scan() {
		text := s.Text()
		logger.Log.Errorf("ulimit:\n %s", text)
	}

	if err := cmd.Wait(); err != nil {
		return 4, err
	}
	return 0, nil
}

func CmdAutoBuild(rootPath string) {
	var (
		err    error
		stderr bytes.Buffer
	)

	runMutex.Lock()
	if isBuilding {
		runMutex.Unlock()
		logger.Log.Info("Build already in progress, skipping...")
		return
	}
	isBuilding = true
	runMutex.Unlock()

	defer func() {
		runMutex.Lock()
		isBuilding = false
		runMutex.Unlock()
	}()

	//for install
	install_cmd := exec.Command("go", "install", "-v")
	install_cmd.Stdout = os.Stdout
	install_cmd.Stderr = os.Stderr
	install_cmd.Env = append(os.Environ(), "GOGC=off")
	err = install_cmd.Run()
	if err != nil {
		logger.Log.Errorf("Intall failed: %s", stderr.String())
		return
	}

	// Check if rebuild is necessary using smart cache
	if !buildCache.ShouldRebuild(rootPath, "go", conf.Ext) {
		logger.Log.Info("No changes detected, skipping build")
		return
	}

	// Start performance monitoring
	stats := monitor.StartBuild()
	defer stats.EndBuild()

	logger.Log.Info("Starting Go build process...")
	logger.Log.Infof("System info: %s", monitor.GetSystemInfo())

	// Change to project directory
	if err := os.Chdir(rootPath); err != nil {
		logger.Log.Errorf("Failed to change directory to %s: %s", rootPath, err)
		return
	}

	rootPath = filepath.ToSlash(rootPath)
	appName := path.Base(rootPath)
	if runtime.GOOS == "windows" {
		appName += ".exe"
	}

	// Build arguments
	args := []string{"build", "-o", appName}
	buildLDFlags = strings.TrimSpace(buildLDFlags)
	if buildLDFlags != "" {
		args = append(args, "-ldflags", buildLDFlags)
	}

	// Execute build command
	buildCmd := exec.Command("go", args...)
	buildCmd.Env = append(os.Environ(), "GOGC=off")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = &stderr

	err = buildCmd.Run()
	if err != nil {
		logger.Log.Errorf("Build failed: %s", stderr.String())
		return
	}

	// Mark build as complete in cache
	buildCache.MarkBuildComplete("go")

	logger.Log.Success("Go build completed successfully")

	CmdRestart(rootPath)
}

func CmdRestart(rootPath string) {
	Kill()
	go CmdStart(rootPath)
}

func CmdStart(rootPath string) {
	runMutex.Lock()
	defer runMutex.Unlock()

	if err := os.Chdir(rootPath); err != nil {
		logger.Log.Errorf("Failed to change directory to %s: %s", rootPath, err)
		return
	}

	appName := path.Base(rootPath)
	if runtime.GOOS == "windows" {
		appName += ".exe"
	}

	logger.Log.Infof("Starting '%s'...", appName)

	// Ensure executable path is correct
	if !strings.Contains(appName, "./") {
		appName = "./" + appName
	}

	// Check if executable exists
	if !tools.IsFile(appName) {
		logger.Log.Errorf("Executable not found: %s", appName)
		return
	}

	cmd = exec.Command(appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in a goroutine
	go func() {
		if err := cmd.Run(); err != nil {
			logger.Log.Errorf("Application exited with error: %s", err)
		} else {
			logger.Log.Info("Application exited normally")
		}
	}()

	// Give the process a moment to start
	time.Sleep(100 * time.Millisecond)

	logger.Log.Successf("'%s' is running...", appName)

	// Non-blocking send to started channel
	select {
	case started <- true:
	default:
	}
}

func CmdDone(rootPath string) {
	// runMutex.Lock()
	// defer runMutex.Unlock()

	CmdRunBefore(rootPath)
	// time.Sleep(1 * time.Second)

	if conf.EnableRun {

		if tools.IsRustP() {
			CmdAutoBuildRust(rootPath)
		} else if tools.IsGoP() {
			CmdAutoBuild(rootPath)
		} else {
			logger.Log.Info("Invalid language environment")
		}

	}
	// time.Sleep(1 * time.Second)
	CmdRunAfter(rootPath)

}

func initWatcher(rootPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Fatalf("Failed to create watcher: %s", err)
	}

	logger.Log.Info("Initializing file watcher...")

	// Channel for debounced file changes
	fileChanges := make(chan string, 100)
	buildTrigger := make(chan bool, 1)

	// File event processor with debouncing
	go func() {
		changedFiles := make(map[string]time.Time)
		ticker := time.NewTicker(time.Duration(conf.Frequency) * time.Second)
		defer ticker.Stop()

		// Config reload ticker (check every 5 seconds)
		configTicker := time.NewTicker(5 * time.Second)
		defer configTicker.Stop()

		for {
			select {
			case filename := <-fileChanges:
				changedFiles[filename] = time.Now()

			case <-ticker.C:
				if len(changedFiles) > 0 {
					// Clear the map and trigger build
					fileCount := len(changedFiles)
					changedFiles = make(map[string]time.Time)

					logger.Log.Infof("Detected changes in %d file(s), triggering rebuild...", fileCount)

					// Non-blocking send to build trigger
					select {
					case buildTrigger <- true:
					default:
						// Build already queued
					}
				}

				// Update ticker frequency if config changed
				runMutex.RLock()
				currentFreq := conf.Frequency
				runMutex.RUnlock()

				if ticker.C != nil {
					ticker.Stop()
					ticker = time.NewTicker(time.Duration(currentFreq) * time.Second)
				}

			case <-configTicker.C:
				// Check for config file changes
				reloadConfig()
			}
		}
	}()

	// Build processor
	go func() {
		for range buildTrigger {
			CmdDone(rootPath)
		}
	}()

	// File system event processor
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// Skip filtered files
				if isFilterFile(event.Name) {
					continue
				}

				// Only process write and create events
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					// Use improved file change detection
					if hasFileChanged(event.Name) {
						logger.Log.Hintf(colors.Bold("Changed: ")+"%s", event.Name)

						// Send to debouncer
						select {
						case fileChanges <- event.Name:
						default:
							// Channel full, skip this event
							logger.Log.Hintf(colors.Bold("Queue full, skipping: ")+"%s", event.Name)
						}
					} else {
						logger.Log.Hintf(colors.Bold("Skipping: ")+"%s (no change)", event.Name)
					}
				}

			case err := <-watcher.Errors:
				logger.Log.Warnf("Watcher error: %s", err)
			}
		}
	}()

	// Add directories to watcher
	dirs := tools.GetPathDir(rootPath, conf.DirFilter)
	dirs = tools.GetVailDir(dirs, conf.Ext)

	for _, dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			logger.Log.Warnf("Failed to watch directory %s: %s", dir, err)
			logger.Log.Info("Tip: If you see 'too many open files', try: ulimit -n 2048")
		} else {
			logger.Log.Hintf(colors.Bold("Watching: ")+"%s", dir)
		}
	}

	logger.Log.Successf("File watcher initialized, monitoring %d directories", len(dirs))
}

func CmdRun(c *cli.Context) error {
	ShowShortVersionBanner()

	buildLDFlags = c.String("ldflags")

	rootPath, _ := os.Getwd()
	appName := path.Base(rootPath)
	logger.Log.Infof("Using '%s' as 'appname'", appName)

	initWatcher(rootPath)
	CmdDone(rootPath)

	for {
		chanel := make(chan os.Signal)
		signal.Notify(chanel, syscall.SIGINT)
		sig := <-chanel

		if sig == syscall.SIGINT {
			fmt.Println("\n")
			logger.Log.Info(fmt.Sprintf("exit: %s", appName))
			CmdRunExit(rootPath)
			break
		}
	}
	return nil
}

package cmd

import (
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/monitor"
)

func CmdAutoBuildRust(rootPath string) {
	runMutex.Lock()
	if isBuilding {
		runMutex.Unlock()
		logger.Log.Info("Rust build already in progress, skipping...")
		return
	}
	isBuilding = true
	runMutex.Unlock()

	defer func() {
		runMutex.Lock()
		isBuilding = false
		runMutex.Unlock()
	}()

	// Check if rebuild is necessary using smart cache
	// For Rust projects, we need to check .rs files specifically
	rustExt := []string{"rs"}
	if !buildCache.ShouldRebuild(rootPath, "rust", rustExt) {
		logger.Log.Info("No changes detected, skipping build")
		return
	}

	// Start performance monitoring
	stats := monitor.StartBuild()
	defer stats.EndBuild()

	logger.Log.Info("Starting Rust build process...")
	logger.Log.Infof("System info: %s", monitor.GetSystemInfo())

	// Change to project directory
	if err := os.Chdir(rootPath); err != nil {
		logger.Log.Errorf("Failed to change directory to %s: %s", rootPath, err)
		return
	}

	// Execute cargo build
	buildCmd := exec.Command("cargo", "build", "--release")
	buildCmd.Dir = rootPath
	buildCmd.Env = append(os.Environ())
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		logger.Log.Errorf("Rust build failed: %s", err)
		return
	}

	// Verify that the executable was created
	appName := path.Base(rootPath)
	executablePath := "./target/release/" + appName
	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		logger.Log.Errorf("Rust executable not found after build: %s", executablePath)
		logger.Log.Info("This might be due to a mismatch between project name and binary name")
		return
	}

	// Mark build as complete in cache
	buildCache.MarkBuildComplete("rust")

	logger.Log.Success("Rust build completed successfully")

	Kill()
	CmdStartRust(rootPath)
}

func CmdStartRust(rootPath string) {
	runMutex.Lock()
	defer runMutex.Unlock()

	if err := os.Chdir(rootPath); err != nil {
		logger.Log.Errorf("Failed to change directory to %s: %s", rootPath, err)
		return
	}

	appName := path.Base(rootPath)
	logger.Log.Infof("Starting '%s'...", appName)

	// Ensure executable path is correct
	if !strings.Contains(appName, "./") {
		appName = "./target/release/" + appName
	}

	// Check if executable exists
	if _, err := os.Stat(appName); os.IsNotExist(err) {
		logger.Log.Errorf("Rust executable not found: %s", appName)
		logger.Log.Info("Make sure 'cargo build --release' completed successfully")
		return
	}

	cmd = exec.Command(appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process in a goroutine
	go func() {
		if err := cmd.Run(); err != nil {
			logger.Log.Errorf("Rust application exited with error: %s", err)
		} else {
			logger.Log.Info("Rust application exited normally")
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

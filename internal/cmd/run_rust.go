package cmd

import (
	"os"
	"os/exec"
	"path"
	"strings"

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
	if !buildCache.ShouldRebuild(rootPath, "rust", conf.Ext) {
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
	buildCmd.Env = append(os.Environ())
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		logger.Log.Errorf("Rust build failed: %s", err)
		return
	}

	// Mark build as complete in cache
	buildCache.MarkBuildComplete("rust")

	logger.Log.Success("Rust build completed successfully")

	Kill()
	CmdStartRust(rootPath)
}

func CmdStartRust(rootPath string) {

	os.Chdir(rootPath)
	appName := path.Base(rootPath)
	logger.Log.Infof("Restarting '%s'...", appName)

	//start
	if !strings.Contains(appName, "./") {
		appName = "./target/release/" + appName
	}

	cmd = exec.Command(appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	go cmd.Run()

	logger.Log.Successf("'%s' is running...", appName)

	started <- true
}

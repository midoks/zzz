package cmd

import (
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/midoks/zzz/internal/logger"
)

func CmdAutoBuildRust(rootPath string) {
	var (
		err error
	)

	os.Chdir(rootPath)

	logger.Log.Infof("%s...", "cargo build --release")
	cmd := exec.Command("cargo", "build", "--release")
	cmd.Env = append(os.Environ())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()
	if err != nil {
		logger.Log.Error("Failed to build!!")
		return
	}

	logger.Log.Success("Built Successfully!")
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

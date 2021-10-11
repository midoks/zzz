package cmd

import (
	"fmt"
	// "log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/logger/colors"
	"github.com/midoks/zzz/internal/tools"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

var Run = cli.Command{
	Name:        "run",
	Usage:       "Run the application by starting a local development server",
	Description: `Run the application by starting a local development server`,
	Action:      CmdRun,
	Flags: []cli.Flag{
		stringFlag("config, c", "", "Custom configuration file path"),
	},
}

var (
	runMutex sync.Mutex
	conf     *ZZZ
	cmd      *exec.Cmd
	exit     chan bool
)
var eventTime = make(map[string]int64)
var started = make(chan bool)

func init() {
	exit = make(chan bool)
	rootPath, _ := os.Getwd()
	file := rootPath + "/" + Zfile
	conf = new(ZZZ)
	if tools.IsExist(file) {
		content, _ := tools.ReadFile(file)
		yaml.Unmarshal([]byte(content), conf)
	} else {
		conf.DirFilter = []string{".git", ".github", "vendor", ".DS_Store", "tmp"}
		conf.Ext = []string{"go"}
	}
}

// Kill kills the running command process
func Kill() {
	defer func() {
		if e := recover(); e != nil {
			logger.Log.Infof("Kill recover: %s", e)
		}
	}()
	if cmd != nil && cmd.Process != nil {
		// Windows does not support Interrupt
		if runtime.GOOS == "windows" {
			cmd.Process.Signal(os.Kill)
		} else {
			cmd.Process.Signal(os.Interrupt)
		}
		ch := make(chan struct{}, 1)
		go func() {
			cmd.Wait()
			ch <- struct{}{}
		}()

		select {
		case <-ch:
			return
		case <-time.After(10 * time.Second):
			logger.Log.Info("Timeout. Force kill cmd process")
			err := cmd.Process.Kill()
			if err != nil {
				logger.Log.Errorf("Error while killing cmd process: %s", err)
			}
			return
		}
	}
}

func isFilterFile(name string) bool {
	suffix := path.Ext(name)
	if suffix == ".go" {
		return false
	}

	return true
}

func CmdRunBefore(rootPath string) {

	logger.Log.Infof("App Run Before Hook Start")

	for _, sh := range conf.Action.Before {

		tmpFile := rootPath + "/." + tools.Md5(sh) + ".sh"
		tools.WriteFile(tmpFile, sh)
		cmd := exec.Command("sh", []string{"-c", tmpFile}...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			logger.Log.Errorf("Run Before Hook Error: %s", err)
		}
		os.Remove(tmpFile)
	}
	logger.Log.Infof("App Run Before Hook End")

}

func CmdRunAfter(rootPath string) {
	// logger.Log.Infof("App Run After Hook Start")
	for _, sh := range conf.Action.After {

		tmpFile := rootPath + "/." + tools.Md5(sh) + ".sh"
		tools.WriteFile(tmpFile, sh)
		cmd := exec.Command("sh", []string{"-c", tmpFile}...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			logger.Log.Errorf("Run After Hook Error:%v", err)
		}
		os.Remove(tmpFile)
	}
	// logger.Log.Infof("App Run After Hook End")
}

func CmdAutoBuild(rootPath string) {
	os.Chdir(rootPath)
	appName := path.Base(rootPath)

	//build
	args := []string{"build"}
	args = append(args, "-o", appName)
	cmd = exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("CmdDo[sddssd]:", err)
	}

	logger.Log.Success("Built Successfully!")
	CmdRestart(rootPath)
}

func CmdRestart(rootPath string) {
	Kill()
	CmdStart(rootPath)
}

func CmdStart(rootPath string) {

	os.Chdir(rootPath)
	appName := path.Base(rootPath)
	logger.Log.Infof("Restarting '%s'...", appName)

	//start
	if !strings.Contains(appName, "./") {
		appName = "./" + appName
	}

	cmd = exec.Command(appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go cmd.Run()

	logger.Log.Successf("'%s' is running...", appName)

	// started <- true
}

func CmdDone(rootPath string) {

	runMutex.Lock()
	defer runMutex.Unlock()

	CmdRunBefore(rootPath)
	// time.Sleep(1 * time.Second)

	CmdAutoBuild(rootPath)

	// time.Sleep(1 * time.Second)
	CmdRunAfter(rootPath)

}

func initWatcher(rootPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Fatalf("Failed to create watcher: %s", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case e := <-watcher.Events:
				if isFilterFile(e.Name) {
					continue
				}

				isBuild := true

				mt := tools.GetFileModTime(e.Name)
				if t := eventTime[e.Name]; mt == t {
					logger.Log.Hintf(colors.Bold("Skipping: ")+"%s", e.String())
					isBuild = false
				}

				eventTime[e.Name] = mt
				if isBuild {

					scheduleTime := time.Now().Add(1 * time.Second)
					time.Sleep(time.Until(scheduleTime))

					rootPath, _ := os.Getwd()
					go CmdDone(rootPath)
				}
			case err := <-watcher.Errors:
				logger.Log.Warnf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	logger.Log.Info("Initializing watcher...")
	dirs := tools.GetPathDir(rootPath, conf.DirFilter)
	for _, d := range dirs {
		err = watcher.Add(d)
		logger.Log.Hintf(colors.Bold("Watching: ")+"%s", d)
		if err != nil {
			logger.Log.Fatalf("Failed to watch directory: %s", err)
		}
	}
	<-done
}

func CmdRun(c *cli.Context) error {
	ShowShortVersionBanner()

	rootPath, _ := os.Getwd()
	appName := path.Base(rootPath)
	logger.Log.Infof("Using '%s' as 'appname'", appName)

	CmdDone(rootPath)
	initWatcher(rootPath)

	return nil
}

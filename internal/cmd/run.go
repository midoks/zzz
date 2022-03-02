package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"

	"github.com/fsnotify/fsnotify"
	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/logger/colors"
	"github.com/midoks/zzz/internal/tools"
)

var Run = cli.Command{
	Name:        "run",
	Usage:       "Run the application by starting a local development server",
	Description: `Run the application by starting a local development server`,
	Action:      CmdRun,
	Flags: []cli.Flag{
		stringFlag("ldflags, ld", "", "Set the build ldflags. See: https://golang.org/pkg/go/build/"),
	},
}

var (
	runMutex     sync.Mutex
	conf         *ZZZ
	cmd          *exec.Cmd
	exit         chan bool
	buildLDFlags string
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
		conf.DirFilter = []string{".git", ".github", "vendor", ".DS_Store", "tmp", ".bak", ".chk"}
		conf.Ext = []string{"go"}
		conf.Frequency = 3
		conf.EnableRun = true
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
	suffix = strings.Trim(suffix, ".")

	if tools.InArray(suffix, conf.Ext) {
		return false
	}
	return true
}

func CmdRunBefore(rootPath string) {

	logger.Log.Infof("App run before hook start")

	for _, sh := range conf.Action.Before {

		tmpFile := rootPath + "/." + tools.Md5(sh) + ".sh"
		tools.WriteFile(tmpFile, sh)
		cmd := exec.Command("sh", []string{"-c", tmpFile}...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		if err != nil {
			logger.Log.Errorf("Run before hook error: %s", err)
		}
		os.Remove(tmpFile)
	}
	logger.Log.Infof("App run before hook end")

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
			logger.Log.Errorf("Run after hook error:%v", err)
		}
		os.Remove(tmpFile)
	}
	// logger.Log.Infof("App Run After Hook End")
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
	cmdName := "go"

	//for install
	icmd := exec.Command(cmdName, "install", "-v")
	icmd.Stdout = os.Stdout
	icmd.Stderr = os.Stderr
	icmd.Env = append(os.Environ(), "GOGC=off")
	icmd.Run()

	os.Chdir(rootPath)
	appName := path.Base(rootPath)

	if runtime.GOOS == "windows" {
		appName += ".exe"
	}
	//build
	args := []string{"build"}
	args = append(args, "-o", appName)

	buildLDFlags = strings.TrimSpace(buildLDFlags)
	if buildLDFlags != "" {
		args = append(args, "-ldflags", buildLDFlags)
	}

	cmd := exec.Command(cmdName, args...)
	cmd.Env = append(os.Environ(), "GOGC=off")
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		logger.Log.Errorf("Failed to build the application: %s", stderr.String())
		return
	}

	logger.Log.Success("Built Successfully!")
	CmdRestart(rootPath)
}

func CmdRestart(rootPath string) {
	Kill()
	go CmdStart(rootPath)
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

	started <- true
}

func CmdDone(rootPath string) {

	runMutex.Lock()
	defer runMutex.Unlock()

	CmdRunBefore(rootPath)
	// time.Sleep(1 * time.Second)

	if conf.EnableRun {
		CmdAutoBuild(rootPath)
	}

	// time.Sleep(1 * time.Second)
	CmdRunAfter(rootPath)

}

func initWatcher(rootPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Log.Fatalf("Failed to create watcher: %s", err)
	}
	// defer watcher.Close()
	doneRun := make(chan int64)
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
					doneRun <- mt
				}
			case err := <-watcher.Errors:
				logger.Log.Warnf("Watcher error: %s", err.Error()) // No need to exit here
			}
		}
	}()

	var changeTime int64
	c := cron.New()

	go func() {
		for {
			changeTime = <-doneRun
			c.Start()
		}
	}()

	cronSpec := fmt.Sprintf("@every %ds", conf.Frequency)
	c.AddFunc(cronSpec, func() {
		if changeTime > 0 {

			if changeTime+conf.Frequency < time.Now().Unix() {
				rootPath, _ := os.Getwd()
				go CmdDone(rootPath)
				c.Stop()
			}
		}
	})
	c.Start()

	logger.Log.Info("Initializing watcher...")
	dirs := tools.GetPathDir(rootPath, conf.DirFilter)
	dirs = tools.GetVailDir(dirs, conf.Ext)
	for _, d := range dirs {
		// fmt.Println(d)
		err = watcher.Add(d)
		logger.Log.Hintf(colors.Bold("Watching: ")+"%s", d)
		if err != nil {
			logger.Log.Fatalf("Failed to watch directory: %s", err)
		}
	}
	// <-done
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
		<-exit
		runtime.Goexit()
	}
	return nil
}

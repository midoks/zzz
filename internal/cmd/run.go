package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	// "time"

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
)

func isFilterFile(name string) bool {
	suffix := path.Ext(name)
	if suffix == ".go" {
		return false
	}

	return true
}

func CmdRunBefore() {
	rootPath, _ := os.Getwd()
	file := rootPath + "/" + Zfile
	conf := new(ZZZ)
	if tools.IsExist(file) {
		logger.Log.Infof("App Run Before Hook Start")

		content, _ := tools.ReadFile(file)
		yaml.Unmarshal([]byte(content), conf)

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
}

func CmdRunAfter() {
	rootPath, _ := os.Getwd()
	file := rootPath + "/" + Zfile
	conf := new(ZZZ)
	if tools.IsExist(file) {
		content, _ := tools.ReadFile(file)
		yaml.Unmarshal([]byte(content), conf)
		logger.Log.Infof("App Run After Hook Start")
		for _, sh := range conf.Action.After {

			tmpFile := rootPath + "/." + tools.Md5(sh) + ".sh"
			tools.WriteFile(tmpFile, sh)
			cmd := exec.Command("sh", []string{"-c", tmpFile}...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				logger.Log.Errorf("Run After Hook Error:", err)
			}
			os.Remove(tmpFile)
		}
		logger.Log.Infof("App Run After Hook End")
	}
}

func CmdDone() {

	// runMutex.Lock()
	// time.Sleep(5 * time.Second)

	rootPath, _ := os.Getwd()
	appName := path.Base(rootPath)

	CmdRunBefore()

	os.Chdir(rootPath)

	//build
	args := []string{"build"}
	args = append(args, "-o", appName)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("CmdDo[sddssd]:", err)
	}

	//start
	if !strings.Contains(appName, "./") {
		appName = "./" + appName
	}

	cmd = exec.Command(appName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go cmd.Run()
	info := fmt.Sprintf("%s is running.s...", appName)
	fmt.Println(info)

	CmdRunAfter()

	//unlock
	// runMutex.Unlock()
}

func CmdRun(c *cli.Context) error {

	rootPath, _ := os.Getwd()

	file := rootPath + "/" + Zfile
	conf := new(ZZZ)
	if tools.IsExist(file) {
		content, _ := tools.ReadFile(file)
		yaml.Unmarshal([]byte(content), conf)
	} else {
		conf.DirFilter = []string{".git", ".github", "vendor", ".DS_Store", "tmp"}
	}

	appName := path.Base(rootPath)
	logger.Log.Infof("Using '%s' as 'appname'", appName)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)

		logger.Log.Infof("NewWatcher Error:'%v'", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}

				//过滤不需要监控的文件
				if !isFilterFile(ev.Name) {
					log.Println("event name:", ev)
					if ev.Op&fsnotify.Create == fsnotify.Create {
						// log.Println("创建文件:", ev.Name)
						CmdDone()
					}
					if ev.Op&fsnotify.Write == fsnotify.Write {
						// log.Println("写入文件:", ev.Name)
						CmdDone()
					}
					if ev.Op&fsnotify.Remove == fsnotify.Remove {
						// log.Println("删除文件:", ev.Name)
						go CmdDone()
					}
					if ev.Op&fsnotify.Rename == fsnotify.Rename {
						// log.Println("重命名文件:", ev.Name)
						go CmdDone()
					}
					if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
						// log.Println("修改权限 : ", ev.Name)
						go CmdDone()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("watcher.Errors:", err)
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

	CmdDone()
	<-done
	return nil
}

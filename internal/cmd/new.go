package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"

	"github.com/midoks/zzz/internal/tools"
)

var New = cli.Command{
	Name:        "new",
	Usage:       "create zzz configuration file",
	Description: `create zzz configuration file`,
	Action:      CmdNew,
	Flags:       []cli.Flag{},
}

// Note: struct fields must be public in order for unmarshal to
// correctly populate the data.
type ZZZ struct {
	Title     string
	Frequency int64
	DirFilter []string
	Ext       []string
	EnableRun bool
	Action    struct {
		Before []string `yaml:"before"`
		After  []string `yaml:"after"`
	}
	Link string
}

const Zfile = ".zzz.yaml"

func CmdNew(c *cli.Context) error {

	rootPath, _ := os.Getwd()
	file := rootPath + "/" + Zfile
	if tools.IsExist(file) {
		fmt.Println("configuration file is exist!")
	} else {

		conf := ZZZ{}
		conf.Title = "zzz"
		conf.Ext = append(conf.Ext, "go")
		conf.Frequency = 3
		conf.EnableRun = true

		conf.DirFilter = append(conf.DirFilter, "tmp")
		conf.DirFilter = append(conf.DirFilter, ".git")
		conf.DirFilter = append(conf.DirFilter, "public")
		conf.DirFilter = append(conf.DirFilter, "scripts")
		conf.DirFilter = append(conf.DirFilter, "vendor")
		conf.DirFilter = append(conf.DirFilter, "logs")
		conf.DirFilter = append(conf.DirFilter, "templates")

		conf.Action.Before = append(conf.Action.Before, "echo \"zzz start\"")
		conf.Action.After = append(conf.Action.After, "echo \"zzz end\"")
		conf.Link = "https://github.com/midoks/zzz"

		d, err := yaml.Marshal(&conf)
		if err != nil {
			fmt.Println("create configuration file fail!")
			return err
		}

		tools.WriteFile(file, string(d))
		fmt.Println("create configuration file successfully!")
	}

	return nil
}

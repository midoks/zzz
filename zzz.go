package main

import (
	"log"
	"os"

	"github.com/midoks/zzz/internal/cmd"
	"github.com/midoks/zzz/internal/conf"
	"github.com/urfave/cli"
)

const (
	Version = "0.0.2"
	AppName = "zzz"
)

func init() {
	conf.App.Version = Version
	conf.App.Name = AppName
}

func main() {

	app := cli.NewApp()
	app.Name = AppName
	app.Version = Version
	app.Usage = "A simple go realtime develop tool"
	app.Commands = []cli.Command{
		cmd.Run,
		cmd.New,
		cmd.Version,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}

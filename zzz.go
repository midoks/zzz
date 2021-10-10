package main

import (
	"log"
	"os"

	"github.com/midoks/zzz/internal/cmd"
	"github.com/urfave/cli"
)

const (
	Version = "0.0.1"
	AppName = "zzz"
)

func main() {

	app := cli.NewApp()
	app.Name = AppName
	app.Version = Version
	app.Usage = "A simple go realtime develop tool"
	app.Commands = []cli.Command{
		cmd.Run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	cmd.CmdRun(nil)
}

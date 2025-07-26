package main

import (
	"log"
	"os"

	"github.com/urfave/cli"

	"github.com/midoks/zzz/internal/cmd"
	"github.com/midoks/zzz/internal/conf"
)

const (
	Version = "0.0.7"
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
	app.Usage = "A high-performance Go/Rust realtime development tool"
	app.Commands = []cli.Command{
		cmd.Run,
		cmd.New,
		cmd.Version,
		cmd.Status,
		cmd.Optimize,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}
}

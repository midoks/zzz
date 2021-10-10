package cmd

import (
	"fmt"

	"github.com/urfave/cli"
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

func CmdRun(c *cli.Context) error {

	fmt.Println("ee")
	return nil
}

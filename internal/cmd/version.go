package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"text/template"
	"time"

	zconf "github.com/midoks/zzz/internal/conf"
	"github.com/midoks/zzz/internal/logger"
	"github.com/midoks/zzz/internal/logger/colors"
	"github.com/urfave/cli"
)

var Version = cli.Command{
	Name:        "version",
	Usage:       "show env info",
	Description: `show env info`,
	Action:      CmdVersion,
	Flags:       []cli.Flag{},
}

// RuntimeInfo holds information about the current runtime.
type RuntimeInfo struct {
	GoVersion  string
	GOOS       string
	GOARCH     string
	NumCPU     int
	GOPATH     string
	GOROOT     string
	Compiler   string
	ZZZVersion string
}

const verboseVersionBanner string = `%s%s
$$$$$$$$\ $$$$$$$$\ $$$$$$$$\ 
\____$$  |\____$$  |\____$$  |
    $$  /     $$  /     $$  / 
   $$  /     $$  /     $$  /  
  $$  /     $$  /     $$  /   
 $$  /     $$  /     $$  /    
$$$$$$$$\ $$$$$$$$\ $$$$$$$$\ 
\________|\________|\________|  v{{ .ZZZVersion }}%s
%s%s
├── GoVersion : {{ .GoVersion }}
├── GOOS      : {{ .GOOS }}
├── GOARCH    : {{ .GOARCH }}
├── NumCPU    : {{ .NumCPU }}
├── GOPATH    : {{ .GOPATH }}
├── GOROOT    : {{ .GOROOT }}
├── Compiler  : {{ .Compiler }}
└── Date      : {{ Now "Monday, 2 Jan 2006" }}%s
`

const shortVersionBanner string = `
$$$$$$$$\ $$$$$$$$\ $$$$$$$$\ 
\____$$  |\____$$  |\____$$  |
    $$  /     $$  /     $$  / 
   $$  /     $$  /     $$  /  
  $$  /     $$  /     $$  /   
 $$  /     $$  /     $$  /    
$$$$$$$$\ $$$$$$$$\ $$$$$$$$\ 
\________|\________|\________|  v{{ .ZZZVersion }}
`

func CmdVersion(c *cli.Context) error {
	coloredBanner := fmt.Sprintf(verboseVersionBanner, "\x1b[35m", "\x1b[1m",
		"\x1b[0m", "\x1b[32m", "\x1b[1m", "\x1b[0m")
	InitBanner(os.Stdout, bytes.NewBufferString(coloredBanner))
	return nil
}

// InitBanner loads the banner and prints it to output
// All errors are ignored, the application will not
// print the banner in case of error.
func InitBanner(out io.Writer, in io.Reader) {
	if in == nil {
		logger.Log.Fatal("The input is nil")
	}

	banner, err := ioutil.ReadAll(in)
	if err != nil {
		logger.Log.Fatalf("Error while trying to read the banner: %s", err)
	}

	show(os.Stdout, string(banner))
}

// ShowShortVersionBanner prints the short version banner.
func ShowShortVersionBanner() {
	output := colors.NewColorWriter(os.Stdout)
	InitBanner(output, bytes.NewBufferString(colors.MagentaBold(shortVersionBanner)))
}

func show(out io.Writer, content string) {
	t, err := template.New("banner").Funcs(template.FuncMap{"Now": Now}).Parse(content)

	if err != nil {
		logger.Log.Fatalf("Cannot parse the banner template: %s", err)
	}

	err = t.Execute(os.Stdout, RuntimeInfo{
		GetGoVersion(),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		os.Getenv("GOPATH"),
		runtime.GOROOT(),
		runtime.Compiler,
		zconf.App.Version,
	})
	if err != nil {
		logger.Log.Error(err.Error())
	}
}

func GetGoVersion() string {
	return runtime.Version()
}

// Now returns the current local time in the specified layout
func Now(layout string) string {
	return time.Now().Format(layout)
}

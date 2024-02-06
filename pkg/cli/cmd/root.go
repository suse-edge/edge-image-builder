package cmd

import (
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

var appName = filepath.Base(os.Args[0])

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = "Edge Image Builder"

	return app
}

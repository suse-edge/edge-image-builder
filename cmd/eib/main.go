package main

import (
	"log"
	"os"

	"github.com/suse-edge/edge-image-builder/pkg/cli/build"
	"github.com/suse-edge/edge-image-builder/pkg/cli/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cmd.NewApp()
	app.Commands = []*cli.Command{
		cmd.NewBuildCommand(build.Run),
		cmd.NewValidateCommand(build.Validate),
		cmd.NewVersionCommand(build.Version),
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

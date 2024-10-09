package main

import (
	"filecoin-utils/utils"
	"os"

	cliutil "github.com/filecoin-project/lotus/cli/util"
	"github.com/filecoin-project/lotus/build"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

var log = logging.Logger("main")

func main() {
	_ = logging.SetLogLevel("*", "INFO")
	log.Info("Starting filecoin-utils")
	
	local := []*cli.Command{
		utilsCmd,
	}
	
	interactiveDef := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	
	app := &cli.App{
		Name:    "filecoin-utils",
		Usage:   "filecoin-utils CLI",
		Version: string(build.NodeUserVersion()),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "panic-reports",
				EnvVars: []string{"LOTUS_PANIC_REPORT_PATH"},
				Hidden:  true,
				Value:   "~/.lotus", // should follow --repo default
			},
			&cli.BoolFlag{
				// examined in the Before above
				Name:        "color",
				Usage:       "use color in display output",
				DefaultText: "depends on output being a TTY",
			},
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Hidden:  true,
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
			&cli.BoolFlag{
				Name:  "interactive",
				Usage: "setting to false will disable interactive functionality of commands",
				Value: interactiveDef,
			},
			&cli.BoolFlag{
				Name:  "force-send",
				Usage: "if true, will ignore pre-send checks",
			},
			cliutil.FlagVeryVerbose,
		},
		After: func(c *cli.Context) error {
			if r := recover(); r != nil {
				// Generate report in LOTUS_PATH and re-raise panic
				build.GenerateNodePanicReport(c.String("panic-reports"), c.String("repo"), c.App.Name)
				panic(r)
			}
			return nil
		},
		Commands: local,
	}

	if err := app.Run(os.Args); err != nil {
		log.Warn(err)
		return
	}
}

var utilsCmd = &cli.Command{
	Name:  "utils",
	Usage: "The extension interface to the filecoin browser project.",
	Subcommands: []*cli.Command{
		utils.ExAddressTransformationCmd,
		utils.ChainExCmd,
		utils.MinerExCmd,
		utils.PowerExCmd,
		utils.AddressTypeCmd,
	},
}
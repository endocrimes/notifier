package main

import (
	"os"
	"sort"

	"github.com/endocrimes/endobot/internal/commands"
	"github.com/hashicorp/go-hclog"
	"github.com/urfave/cli/v2"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "endobot",
		Level: hclog.LevelFromString("DEBUG"),
	})

	app := &cli.App{
		Name:  "endobot",
		Usage: "A telegram bot for dealing with my laziness",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "start the telegram bot",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name: "telegram-token",
						EnvVars: []string{
							"TELEGRAM_BOT_TOKEN",
						},
						Usage:    "Bot token to use for interacting with the telegram api",
						Required: true,
					},
					&cli.StringFlag{
						Name: "jwt-secret",
						EnvVars: []string{
							"ENDOBOT_JWT_SECRET",
						},
						Usage:    "Secret key that should be used to sign api tokens",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "listen-addr",
						Usage: "The address that the HTTP API should listen to",
						Value: ":8080",
					},
				},
				Action: func(c *cli.Context) error {
					return commands.RunCommand(c, logger)
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		logger.Error("endobot failed", "error", err)
	}
}

package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/endocrimes/endobot/internal/api"
	"github.com/endocrimes/endobot/internal/bot"
	"github.com/endocrimes/endobot/internal/tokensigner/jwt"
	jwtlib "github.com/gbrlsnchs/jwt/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/hashicorp/go-hclog"
	"github.com/urfave/cli/v2"
)

const (
	Version = "0.0.1"
)

func RunCommand(c *cli.Context, logger hclog.Logger) error {
	logger.Info("Starting", "version", Version)
	telegramToken := c.String("telegram-token")
	if telegramToken == "" {
		return fmt.Errorf("missing required argument: telegram-token")
	}

	jwtSecretStr := c.String("jwt-secret")
	if jwtSecretStr == "" {
		return fmt.Errorf("missing required argument: jwt-secret")
	}
	jwtSecret := jwtlib.NewHS256([]byte(jwtSecretStr))
	signer := &jwt.TokenSigner{Secret: jwtSecret}

	tg, err := tgbotapi.NewBotAPI(telegramToken)
	if err != nil {
		return fmt.Errorf("telegram setup failed: %v", err)
	}
	logger.Info("telegram initialized", "bot_username", tg.Self.UserName)

	shutdownCtx, cancelFn := context.WithCancel(context.Background())
	errCh := make(chan error, 2)

	bot := bot.New(logger, tg, signer)
	go func() {
		err := bot.Run(shutdownCtx)
		if err != nil {
			errCh <- err
		}
	}()

	srv := api.NewServer(logger, bot, signer)
	go func() {
		err := srv.Start(shutdownCtx, c.String("listen-addr"))
		if err != nil {
			errCh <- err
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		cancelFn()
	}()

	select {
	case err := <-errCh:
		cancelFn()
		return err
	case <-shutdownCtx.Done():
		return nil
	}
}

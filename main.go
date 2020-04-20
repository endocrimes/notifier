package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/endocrimes/notifier/internal/api"
	"github.com/endocrimes/notifier/internal/bot"
	"github.com/endocrimes/notifier/internal/tokensigner"
	"github.com/endocrimes/notifier/internal/tokensigner/jwt"
	jwtlib "github.com/gbrlsnchs/jwt/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/hashicorp/go-hclog"
)

const (
	Version = "0.0.1"
)

func loadSignerFromEnv(logger hclog.Logger) tokensigner.TokenSigner {
	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		logger.Warn("missing configuration for signatures, using developer mode.")
		jwtSecretStr = "D3V3L0P3R_M0D3"
	}

	jwtSecret := jwtlib.NewHS256([]byte(jwtSecretStr))
	return &jwt.TokenSigner{Secret: jwtSecret}
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "notifier",
		Level: hclog.LevelFromString("DEBUG"),
	})

	logger.Info("Starting", "version", Version)

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		logger.Error("missing required configuration: TELEGRAM_BOT_TOKEN")
		os.Exit(1)
	}

	tg, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to initialize telegram library", "error", err)
		os.Exit(1)
	}
	logger.Info("telegram initialized", "bot_username", tg.Self.UserName)

	shutdownCtx, cancelFn := context.WithCancel(context.Background())
	errCh := make(chan error, 2)

	signer := loadSignerFromEnv(logger)

	bot := bot.New(logger, tg, signer)
	go func() {
		err := bot.Run(shutdownCtx)
		if err != nil {
			errCh <- err
		}
	}()

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	srv := api.NewServer(logger, bot, signer)
	go func() {
		err := srv.Start(shutdownCtx, listenAddr)
		if err != nil {
			errCh <- err
		}
	}()

	go func() {
		for {
			select {
			case err := <-errCh:
				logger.Error("Shutting down due to error", "error", err)
				cancelFn()
			case <-shutdownCtx.Done():
				logger.Debug("Shutting down")
			}
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		cancelFn()
	}()

	<-shutdownCtx.Done()
}

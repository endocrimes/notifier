package bot

import (
	"context"
	"fmt"

	"github.com/endocrimes/notifier/internal/tokensigner"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/hashicorp/go-hclog"
)

type Bot struct {
	tokenSigner tokensigner.TokenSigner
	tg          *tgbotapi.BotAPI
	logger      hclog.Logger
}

func New(logger hclog.Logger, tg *tgbotapi.BotAPI, ts tokensigner.TokenSigner) *Bot {
	return &Bot{
		tokenSigner: ts,
		logger:      logger,
		tg:          tg,
	}
}

func (b *Bot) processCommand(cmd string, update tgbotapi.Update) {
	switch cmd {
	case "register":
		b.logger.Info("processing command", "command", cmd, "user_id", update.Message.From.ID)
		tokenBytes, err := b.tokenSigner.GenerateToken(update.Message.Chat, update.Message.From)
		if err != nil {
			b.logger.Error("failed to generate token", "user", update.Message.From.UserName, "error", err)
			b.tg.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Oops! an error occured :("))
		}
		token := string(tokenBytes)
		b.tg.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Your token is: %s", token)))
	default:
		b.tg.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Unknown command: %s", cmd)))
	}
}

func (b *Bot) Notify(chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	msg.DisableNotification = false
	_, err := b.tg.Send(msg)
	return err
}

func (b *Bot) Run(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := b.tg.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.Message == nil {
				// Not sure when message is nil (maybe updates?), but guarding against
				// it here.
				continue
			}

			if !update.Message.IsCommand() {
				b.logger.Trace("unrecognized message", "user_id", update.Message.From.ID)
				continue
			}

			cmd := update.Message.Command()
			b.processCommand(cmd, update)
		}
	}
}

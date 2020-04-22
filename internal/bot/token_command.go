package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var tokenCmd = &botCommand{
	Alias: "token",
	RunFunc: func(ctx context.Context, b *Bot, update tgbotapi.Update) error {
		tokenBytes, err := b.tokenSigner.GenerateToken(update.Message.Chat, update.Message.From)
		if err != nil {
			return err
		}
		token := string(tokenBytes)
		b.tg.Send(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Your token is: %s", token)))
		return nil
	},
}

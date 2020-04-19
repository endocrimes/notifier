package tokensigner

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TokenSigner interface {
	GenerateToken(chat *tgbotapi.Chat, user *tgbotapi.User) ([]byte, error)
	VerifyToken(token []byte) (int64, error)
}

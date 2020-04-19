package jwt

import (
	"fmt"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	uuid "github.com/satori/go.uuid"
)

type ChatToken struct {
	jwt.Payload
	ChatID int64 `json:"chat_id"`
}

type TokenSigner struct {
	Secret *jwt.HMACSHA
}

func (t *TokenSigner) GenerateToken(chat *tgbotapi.Chat, user *tgbotapi.User) ([]byte, error) {
	now := time.Now()
	pl := ChatToken{
		Payload: jwt.Payload{
			Issuer:         "Terrible Systems",
			Subject:        user.UserName,
			ExpirationTime: jwt.NumericDate(now.Add(12 * 30 * 24 * time.Hour)),
			IssuedAt:       jwt.NumericDate(now),
			JWTID:          uuid.NewV4().String(),
		},
		ChatID: chat.ID,
	}

	return jwt.Sign(pl, t.Secret)
}

func (t *TokenSigner) VerifyToken(token []byte) (int64, error) {
	var ct ChatToken
	_, err := jwt.Verify(token, t.Secret, &ct)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	if ct.ExpirationTime.Before(now) {
		return 0, fmt.Errorf("Token has expired")
	}

	return ct.ChatID, nil
}

package bot

import (
	"context"

	"github.com/endocrimes/endobot/internal/tokensigner"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/hashicorp/go-hclog"
)

type botCommand struct {
	Alias   string
	RunFunc func(ctx context.Context, bot *Bot, update tgbotapi.Update) error
}

type Bot struct {
	tokenSigner tokensigner.TokenSigner
	tg          *tgbotapi.BotAPI
	logger      hclog.Logger
	commands    map[string]*botCommand
}

func New(logger hclog.Logger, tg *tgbotapi.BotAPI, ts tokensigner.TokenSigner) *Bot {
	b := &Bot{
		tokenSigner: ts,
		logger:      logger,
		tg:          tg,
		commands:    make(map[string]*botCommand),
	}

	cmds := []*botCommand{
		tokenCmd,
	}
	for _, cmd := range cmds {
		b.commands[cmd.Alias] = cmd
	}

	return b
}

func (b *Bot) processCommand(cmd string, update tgbotapi.Update) {
	b.logger.Info("processing command", "command", cmd, "user_id", update.Message.From.ID)
	impl, ok := b.commands[cmd]
	if !ok {
		b.logger.Trace("command not found", "command", cmd)
		b.tg.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, I didn't recognize that command."))
		return
	}

	// TODO: Set a timeout.
	ctx := context.Background()
	err := impl.RunFunc(ctx, b, update)
	if err != nil {
		b.logger.Error("failed to execute command", "error", err, "command", cmd)
		b.tg.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Sorry, something went wrong :("))
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

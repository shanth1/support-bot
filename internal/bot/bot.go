package bot

import (
	"fmt"
	"strings"

	"github.com/shanth1/support-bot/internal/storage"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	bot     *tele.Bot
	adminID int64
	store   *storage.Storage
}

func New(token string, adminID int64, store *storage.Storage) (*Bot, error) {
	b, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10},
	})
	if err != nil {
		return nil, err
	}

	return &Bot{bot: b, adminID: adminID, store: store}, nil
}

func (b *Bot) Start() {
	b.registerHandlers()
	b.bot.Start()
}

func (b *Bot) registerHandlers() {
	b.bot.Handle(tele.OnText, b.handleMessage)
	b.bot.Handle(tele.OnPhoto, b.handleMessage)
}

func (b *Bot) handleMessage(c tele.Context) error {
	if c.Chat().ID == b.adminID {
		return b.handleAdminReply(c)
	}

	if c.Chat().Type == tele.ChatPrivate {
		return b.forwardToAdmin(c)
	}

	return nil
}

func (b *Bot) forwardToAdmin(c tele.Context) error {
	fwd, err := b.bot.Forward(tele.ChatID(b.adminID), c.Message())
	if err != nil {
		return err
	}

	_ = b.store.SaveMessage(fwd.ID, c.Sender().ID)

	return c.Send("✅ Ваше сообщение передано поддержке.")
}

func (b *Bot) handleAdminReply(c tele.Context) error {
	if !c.Message().IsReply() {
		return nil
	}

	userID, err := b.store.GetUserByMsg(c.Message().ReplyTo.ID)
	if err != nil {
		return c.Reply("❌ Не нашел кому ответить (возможно, старое сообщение)")
	}

	_, err = b.bot.Copy(tele.ChatID(userID), c.Message())
	if err != nil {
		return c.Reply("❌ Ошибка доставки: " + err.Error())
	}

	return c.Reply("✅ Отправлено пользователю")
}

func (b *Bot) SendNotification(project, msgType, name, email, text string, topicID int) error {
	formatted := fmt.Sprintf("🚀 <b>%s</b> [%s]\n\n<b>От:</b> %s (%s)\n\n%s",
		strings.ToUpper(project), msgType, name, email, text)

	opts := &tele.SendOptions{
		ParseMode: tele.ModeHTML,
		ThreadID:  topicID,
	}

	_, err := b.bot.Send(tele.ChatID(b.adminID), formatted, opts)
	return err
}

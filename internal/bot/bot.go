package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/shanth1/gotools/log"
	"github.com/shanth1/gotools/logkeys"
	"github.com/shanth1/support-bot/internal/config"
	"github.com/shanth1/support-bot/internal/storage"
	tele "gopkg.in/telebot.v3"
)

type SupportBot struct {
	bot   *tele.Bot
	store *storage.Storage
	cfg   *config.Config
	log   log.Logger
}

func New(cfg *config.Config, store *storage.Storage, l log.Logger) (*SupportBot, error) {
	pref := tele.Settings{
		Token:  cfg.Bot.Token,
		Poller: &tele.LongPoller{Timeout: cfg.Bot.PollerTimeout},
		OnError: func(err error, c tele.Context) {
			l.Error().
				Err(err).
				Str(logkeys.Component, "telebot").
				Msg("Telegram error")
		},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, err
	}

	return &SupportBot{
		bot:   b,
		store: store,
		cfg:   cfg,
		log:   l,
	}, nil
}

func (b *SupportBot) Run(ctx context.Context, shutdownCtx context.Context) error {
	b.registerHandlers()

	go b.bot.Start()
	b.log.Info().Str(logkeys.Service, b.bot.Me.Username).Msg("bot started")

	<-ctx.Done()
	b.log.Info().Msg("stopping bot...")

	stopped := make(chan struct{})
	go func() {
		b.bot.Stop()
		close(stopped)
	}()

	// Ждем либо пока бот сам остановится, либо пока истечет shutdownCtx
	select {
	case <-stopped:
		b.log.Info().Msg("Bot stopped gracefully")
	case <-shutdownCtx.Done():
		b.log.Warn().Msg("Bot stopping timed out (shutdownCtx expired)")
	}

	return nil
}

func (b *SupportBot) registerHandlers() {
	b.bot.Handle("/start", func(c tele.Context) error {
		return c.Send(b.cfg.Messages.Start)
	})

	b.bot.Handle(tele.OnText, b.handleMessage)
	b.bot.Handle(tele.OnPhoto, b.handleMessage)
	b.bot.Handle(tele.OnDocument, b.handleMessage)
	b.bot.Handle(tele.OnVideo, b.handleMessage)
}

func (b *SupportBot) handleMessage(c tele.Context) error {
	if c.Chat().ID == b.cfg.Bot.AdminGroupID {
		return b.handleAdminReply(c)
	}
	if c.Chat().Type == tele.ChatPrivate {
		return b.forwardToAdmin(c)
	}
	return nil
}

func (b *SupportBot) forwardToAdmin(c tele.Context) error {
	sendOpts := &tele.SendOptions{ThreadID: b.cfg.Bot.TopicID}

	userLink := fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", c.Sender().ID, c.Sender().FirstName)
	headerText := fmt.Sprintf("%s %s\nID: <code>%d</code>",
		b.cfg.Messages.AdminNotificationHeader, userLink, c.Sender().ID)

	headerMsg, err := b.bot.Send(tele.ChatID(b.cfg.Bot.AdminGroupID), headerText, sendOpts, tele.ModeHTML)
	if err != nil {
		b.log.Error().Err(err).Msg("Failed to send header to admin")
		return c.Send(b.cfg.Messages.ErrorGeneric)
	}

	forwardedMsg, err := b.bot.Forward(tele.ChatID(b.cfg.Bot.AdminGroupID), c.Message(), sendOpts)
	if err != nil {
		b.log.Error().Err(err).Msg("Failed to forward message")
		return c.Send(b.cfg.Messages.ErrorGeneric)
	}

	_ = b.store.SaveRoute(context.Background(), headerMsg.ID, c.Sender().ID)
	_ = b.store.SaveRoute(context.Background(), forwardedMsg.ID, c.Sender().ID)

	return c.Send(b.cfg.Messages.UserSentOk)
}

func (b *SupportBot) handleAdminReply(c tele.Context) error {
	if !c.Message().IsReply() {
		return nil
	}

	userID, err := b.store.GetUserByAdminMsg(context.Background(), c.Message().ReplyTo.ID)
	if err != nil {
		b.log.Debug().
			Int("admin_msg_id", c.Message().ReplyTo.ID).
			Err(err).
			Msg("Route not found. Ignoring message (possibly belongs to another bot)")

		return nil
	}

	_, err = b.bot.Copy(tele.ChatID(userID), c.Message())

	if err != nil {
		errText := err.Error()

		b.log.Error().
			Err(err).
			Int64(logkeys.UserID, userID).
			Str("tele_error", errText).
			Msg("Failed to copy message to user")

		isBlocked := strings.Contains(strings.ToLower(errText), "blocked") ||
			strings.Contains(strings.ToLower(errText), "forbidden") ||
			strings.Contains(strings.ToLower(errText), "deactivated") ||
			strings.Contains(strings.ToLower(errText), "initiated")

		if isBlocked {
			return c.Reply(b.cfg.Messages.ErrorUserBlocked)
		}

		return c.Reply(b.cfg.Messages.ErrorCopyFailed)
	}

	return c.Reply(b.cfg.Messages.AdminReplySent)
}

func (b *SupportBot) SendNotification(title, message string) error {
	text := fmt.Sprintf("🔔 <b>%s</b>\n\n%s", title, message)
	_, err := b.bot.Send(tele.ChatID(b.cfg.Bot.AdminGroupID), text, &tele.SendOptions{
		ThreadID:  b.cfg.Bot.TopicID,
		ParseMode: tele.ModeHTML,
	})
	return err
}

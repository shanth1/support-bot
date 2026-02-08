package bot

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/shanth1/support-bot/internal/config"
	"github.com/shanth1/support-bot/internal/storage"
	tele "gopkg.in/telebot.v3"
)

type Bot struct {
	bot     *tele.Bot
	adminID int64
	store   *storage.Storage
	cfg     *config.Config
	logger  *slog.Logger
}

func New(cfg *config.Config, store *storage.Storage, logger *slog.Logger) (*Bot, error) {
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10},
	})
	if err != nil {
		return nil, err
	}

	return &Bot{
		bot:     b,
		adminID: cfg.AdminGroupID,
		store:   store,
		cfg:     cfg,
		logger:  logger,
	}, nil
}

func (b *Bot) Start() {
	b.registerHandlers()
	b.logger.Info("СЕРВИС ПОДКЛЮЧЕН",
		"bot_name", b.bot.Me.FirstName,
		"username", "@"+b.bot.Me.Username,
	)
	b.bot.Start()
}

func (b *Bot) registerHandlers() {
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle(tele.OnText, b.handleMessage)
	b.bot.Handle(tele.OnPhoto, b.handleMessage)
}

func (b *Bot) handleStart(c tele.Context) error {
	projectID := c.Data()
	if projectID == "" {
		projectID = "default"
	}
	pCfg := b.cfg.GetProject(projectID)

	_ = b.store.SetUserProject(c.Sender().ID, pCfg.ID)

	return c.Send(pCfg.Greeting)
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
	projectID, _ := b.store.GetUserProject(c.Sender().ID)
	pCfg := b.cfg.GetProject(projectID)

	opts := &tele.SendOptions{
		ThreadID:  pCfg.TopicID,
		ParseMode: tele.ModeHTML,
	}

	header := fmt.Sprintf("👤 <b>%s</b>\n📂 Проект: %s\n\n", c.Sender().FirstName, pCfg.Name)

	var msg *tele.Message
	var err error

	if c.Message().Text != "" {
		msg, err = b.bot.Send(tele.ChatID(b.adminID), header+c.Message().Text, opts)
	} else {
		b.bot.Send(tele.ChatID(b.adminID), header, opts)
		msg, err = b.bot.Forward(tele.ChatID(b.adminID), c.Message(), opts)
	}

	if err == nil {
		_ = b.store.SaveMessage(msg.ID, c.Sender().ID)
		return c.Send(b.cfg.Messages.UserSentOk)
	}

	return err
}

func (b *Bot) handleAdminReply(c tele.Context) error {
	if !c.Message().IsReply() {
		return nil
	}

	userID, err := b.store.GetUserByMsg(c.Message().ReplyTo.ID)
	if err != nil {
		return c.Reply(b.cfg.Messages.ErrorNotFound)
	}

	_, err = b.bot.Copy(tele.ChatID(userID), c.Message())
	if err != nil {
		return c.Reply(b.cfg.Messages.ErrorUserBlocked)
	}

	return c.Reply(b.cfg.Messages.AdminSentOk)
}

func (b *Bot) SendNotification(project, msgType, name, email, message string, meta map[string]string) error {
	pCfg := b.cfg.GetProject(project)

	text := renderTemplate(b.cfg.Templates.AdminNotification, map[string]interface{}{
		"ProjectName": pCfg.Name,
		"Type":        msgType,
		"Name":        name,
		"Email":       email,
		"Message":     message,
		"Meta":        meta,
	})

	_, err := b.bot.Send(tele.ChatID(b.adminID), text, &tele.SendOptions{
		ThreadID:  pCfg.TopicID,
		ParseMode: tele.ModeHTML,
	})
	return err
}

func renderTemplate(tmplStr string, data interface{}) string {
	tmpl, err := template.New("msg").Parse(tmplStr)
	if err != nil {
		return "Template Error: " + err.Error()
	}
	var buf bytes.Buffer
	_ = tmpl.Execute(&buf, data)
	return buf.String()
}

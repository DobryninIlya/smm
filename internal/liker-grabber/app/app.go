package app

import (
	"context"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
	"smm_media/internal/liker-grabber/config"
	"smm_media/internal/liker-grabber/model"
	"smm_media/internal/liker-grabber/tg-assembly"
)

type App struct {
	log      *logrus.Logger
	messages chan *tg.Message
	accounts map[string]*model.Phone
	config   *config.Config
	ctx      context.Context
}

func NewApp(ctx context.Context, log *logrus.Logger, messages chan *tg.Message, config *config.Config) *App {
	return &App{
		log:      log,
		messages: messages,
		ctx:      ctx,
		accounts: config.Phone,
		config:   config,
	}
}

func (a *App) Start() {
	tgAssembly := tg_assembly.NewTelegramAssembly(a.ctx, a.messages, a.log, a.accounts, a.config)
	tgAssembly.Start()
}

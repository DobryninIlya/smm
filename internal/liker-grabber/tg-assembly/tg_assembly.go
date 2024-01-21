package tg_assembly

import (
	"context"
	"fmt"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
	"smm_media/internal/liker-grabber/config"
	"smm_media/internal/liker-grabber/model"
	"sync"
)

type TelegramAssembly struct {
	apiId    int
	apiHash  string
	messages chan *tg.Message
	chats    []string
	clients  []Client
	log      *logrus.Logger
	accounts map[string]*model.Phone
	config   *config.Config
	ctx      context.Context
}

func NewTelegramAssembly(ctx context.Context, messages chan *tg.Message, log *logrus.Logger, accounts map[string]*model.Phone, config *config.Config, mainConfig *config.MainConfig) *TelegramAssembly {
	return &TelegramAssembly{
		messages: messages,
		ctx:      ctx,
		log:      log,
		accounts: accounts,
		config:   config,
		apiHash:  mainConfig.ApiHash,
		apiId:    mainConfig.ApiID,
	}
}

func (t *TelegramAssembly) Start() {
	t.log.Info("TelegramAssembly started")
	wg := sync.WaitGroup{}
	for phone, phoneProxy := range t.accounts {
		login := phone
		proxy := phoneProxy.Proxy
		like := phoneProxy.Like
		parse := phoneProxy.Parse
		comment := phoneProxy.Comment
		chatLinks := phoneProxy.ChatLinks
		name := phoneProxy.Name
		lastname := phoneProxy.LastName
		about := phoneProxy.About
		comments := phoneProxy.Comments
		clientName := phone + " | " + name + " " + lastname
		wg.Add(1)
		go func() {
			t.log.Info("Creating client for login: ", login)
			client, err := NewClient(t.ctx, "+"+login, proxy, t.log, chatLinks, t.messages, like, parse, comment, t.config, t.apiHash, t.apiId, clientName, comments)
			if err != nil {
				t.log.Fatal(err)
			}
			err = client.StartWaiter(t.ctx, name, lastname, about)
			if err != nil {
				t.log.Logf(
					logrus.ErrorLevel,
					"Ошибка запуска клиента %v: %v \nВероятно, вам следует добавить бота заново через скрипт авторизации",
					login,
					err.Error(),
				)
			}
			t.clients = append(t.clients, *client)
			wg.Done()
		}()

	}
	defer close(t.messages)
	go t.processMessages()
	wg.Wait()
}

func (t *TelegramAssembly) processMessages() {
	for {
		select {
		case message := <-t.messages:
			//case <-t.messages:

			//t.log.Info("Message received: ", message.Message)
			fmt.Println("Message received: ", message.Message)
		case <-t.ctx.Done():
			return
		}
	}
}

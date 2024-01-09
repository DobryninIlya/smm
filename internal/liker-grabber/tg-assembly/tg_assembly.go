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

func NewTelegramAssembly(ctx context.Context, messages chan *tg.Message, log *logrus.Logger, accounts map[string]*model.Phone, config *config.Config) *TelegramAssembly {
	return &TelegramAssembly{
		messages: messages,
		ctx:      ctx,
		log:      log,
		accounts: accounts,
		config:   config,
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
		chatLinks := phoneProxy.ChatLinks
		fmt.Println(chatLinks)
		wg.Add(1)
		go func() {
			t.log.Info("Creating client for login: ", login)
			client, err := NewClient(t.ctx, "+"+login, proxy, t.log, chatLinks, t.messages, like, parse, t.config)
			if err != nil {
				t.log.Fatal(err)
			}
			err = client.StartWaiter()
			if err != nil {
				t.log.Fatal(err)
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

			t.log.Info("Message received: ", message.Message)
		case <-t.ctx.Done():
			return
		}
	}
}

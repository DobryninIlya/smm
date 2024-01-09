package tg_assembly

import (
	"bufio"
	"context"
	"fmt"
	pebbledb "github.com/cockroachdb/pebble"
	boltstor "github.com/gotd/contrib/bbolt"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/contrib/pebble"
	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
	lj "gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"path/filepath"
	"smm_media/internal/liker-grabber/config"
	"strconv"
	"strings"
	"time"
)

const (
	sessionPath = "session"
	api_hash    = "8da85b0d5bfe62527e5b244c209159c3"
	api_id      = 2496
)

var successCounter int

type Proxy struct {
	IP       string
	Port     string
	Login    string
	Password string
}

type Client struct {
	app             *telegram.Client
	banned          bool
	proxy           Proxy
	phone           string
	appID           int
	appHash         string
	sessionDir      string
	sender          *message.Sender
	waiter          *floodwait.Waiter
	flow            auth.Flow
	log             *logrus.Logger
	lg              *zap.Logger
	updatesRecovery *updates.Manager
	api             *tg.Client
	ctx             context.Context
	chats           []string
	addedChats      []string
	config          *config.Config
}

func (c *Client) setSessionDir() {
	c.sessionDir = filepath.Join("session", sessionFolder(c.phone))
	if err := os.MkdirAll(c.sessionDir, 0700); err != nil {
		log.Println(err)
	}
}

func NewClient(ctx context.Context, phone, proxyParams string, log *logrus.Logger, chats []string, messages chan *tg.Message, like, parse bool, config *config.Config) (*Client, error) {
	client := &Client{
		banned:  false,
		proxy:   ParseProxy(proxyParams),
		waiter:  &floodwait.Waiter{},
		phone:   phone,
		log:     log,
		appID:   api_id,
		appHash: api_hash,
		ctx:     ctx,
		chats:   chats,
		config:  config,
	}
	client.setSessionDir()
	logFilePath := filepath.Join(client.sessionDir, "log.jsonl")
	log.Printf("Storing session in %s, logs in %s\n", client.sessionDir, logFilePath)
	logWriter := zapcore.AddSync(&lj.Logger{
		Filename:   logFilePath,
		MaxBackups: 3,
		MaxSize:    1, // megabytes
		MaxAge:     7, // days
	})
	logCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		logWriter,
		zap.DebugLevel,
	)
	lg := zap.New(logCore)
	defer func() { _ = lg.Sync() }()
	client.lg = lg
	sessionStorage := &telegram.FileSessionStorage{
		Path: filepath.Join(client.sessionDir, "session.json"),
	}
	db, err := pebbledb.Open(filepath.Join(client.sessionDir, "peers.pebble.db"), &pebbledb.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "create pebble storage")
	}
	peerDB := pebble.NewPeerStorage(db)
	lg.Info("Storage", zap.String("path", client.sessionDir))

	dispatcher := tg.NewUpdateDispatcher()
	updateHandler := storage.UpdateHook(dispatcher, peerDB)

	boltdb, err := bbolt.Open(filepath.Join(client.sessionDir, "updates.bolt.db"), 0666, nil)
	if err != nil {
		return nil, errors.Wrap(err, "create bolt storage")
	}
	updatesRecovery := updates.New(updates.Config{
		Handler: updateHandler, // using previous handler with peerDB
		Logger:  lg.Named("updates.recovery"),
		Storage: boltstor.NewStateStorage(boltdb),
	})
	client.updatesRecovery = updatesRecovery

	// Handler of FLOOD_WAIT that will automatically retry request.
	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		// Notifying about flood wait.
		lg.Warn("Flood wait", zap.Duration("wait", wait.Duration))
		fmt.Println("Got FLOOD_WAIT. Will retry after", wait.Duration)
	})
	client.waiter = waiter
	// Filling client options.
	sock5, _ := proxy.SOCKS5("tcp", client.proxy.IP+":"+client.proxy.Port, &proxy.Auth{
		User:     client.proxy.Login,
		Password: client.proxy.Password,
	}, proxy.Direct)
	dc := sock5.(proxy.ContextDialer)

	options := telegram.Options{
		Logger:         lg,              // Passing logger for observability.
		SessionStorage: sessionStorage,  // Setting up session sessionStorage to store auth data.
		UpdateHandler:  updatesRecovery, // Setting up handler for updates from server.
		Middlewares: []telegram.Middleware{
			// Setting up FLOOD_WAIT handler to automatically wait and retry request.
			waiter,
			// Setting up general rate limits to less likely get flood wait errors.
			ratelimit.New(rate.Every(time.Millisecond*3000), 3),
		},
		Resolver: dcs.Plain(dcs.PlainOptions{
			Dial: dc.DialContext,
		}),
	}
	client.app = telegram.NewClient(client.appID, client.appHash, options)
	api := client.app.API()
	client.api = api
	client.sender = message.NewSender(api)
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		msg, ok := u.Message.(*tg.Message)
		if ok {
			if parse {
				messages <- msg
			}
			if like {
				peerID, err := peer.NewEntities(e.Users, e.Chats, e.Channels).ExtractPeer(msg.GetPeerID())

				reaction := []tg.ReactionClass{
					&tg.ReactionEmoji{Emoticon: "👍"},
				}
				if err != nil {
					fmt.Println("Ошибка пиров: " + err.Error())
				}
				_, err = client.api.MessagesSendReaction(ctx, &tg.MessagesSendReactionRequest{
					Peer:     peerID,
					MsgID:    msg.ID - 1,
					Reaction: reaction,
				})
				reactions, err := client.api.MessagesGetAvailableReactions(ctx, 0)
				resultReaction, ok := reactions.(*tg.MessagesAvailableReactions)
				if !ok {
					fmt.Println("Ошибка при получении реакций")
					return nil
				}
				if err != nil {
					reaction[0] = &tg.ReactionEmoji{Emoticon: resultReaction.Reactions[0].Reaction}
					_, err = client.sender.To(peerID).Reaction(ctx, msg.ID, reaction...)
				}
				if err != nil {
					fmt.Println("Ошибка постановки реакции на сообщение: " + err.Error() + "\nВ чате " + msg.GetPeerID().String())
				} else {
					successCounter++
					client.log.Println(strconv.Itoa(successCounter) + " Поставил реакцию на сообщение в чате " + msg.GetPeerID().String())
				}
				client.sender.Reply(e, u)
			}
		}
		return nil
	})

	codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
		fmt.Print("Enter code for " + phone + ": ")
		code, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(code), nil
	}
	password := "Nasok174"
	flow := auth.NewFlow(
		auth.Constant(phone, password, auth.CodeAuthenticatorFunc(codePrompt)),
		auth.SendCodeOptions{},
	)
	client.flow = flow
	return client, nil

}

func (c *Client) StartWaiter() error {
	err := c.waiter.Run(c.ctx, func(ctx context.Context) error {
		// Spawning main goroutine.
		if err := c.app.Run(ctx, func(ctx context.Context) error {
			// Perform auth if no session is available.
			if err := c.app.Auth().IfNecessary(ctx, c.flow); err != nil {
				return errors.Wrap(err, "auth")
			}

			// Getting info about current user.
			self, err := c.app.Self(ctx)
			if err != nil {
				return errors.Wrap(err, "call self")
			}

			name := self.FirstName
			if self.Username != "" {
				// Username is optional.
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}
			fmt.Println("Current user:", name)

			c.lg.Info("Login",
				zap.String("first_name", self.FirstName),
				zap.String("last_name", self.LastName),
				zap.String("username", self.Username),
				zap.Int64("id", self.ID),
			)

			// Waiting until context is done.
			fmt.Println("Joining chats")
			c.JoinChats()
			fmt.Println("Listening for updates. Interrupt (Ctrl+C) to stop.")
			return c.updatesRecovery.Run(ctx, c.api, self.ID, updates.AuthOptions{
				IsBot: self.Bot,
				OnStart: func(ctx context.Context) {
					fmt.Println("Update recovery initialized and started, listening for events")
				},
			})
		}); err != nil {
			return errors.Wrap(err, "run")
		}
		return nil
	})
	return err
}

func (c *Client) JoinChats() {
	for _, chat := range c.chats {
		for _, addedChat := range c.addedChats {
			if chat == addedChat {
				continue
			}
		}
		_, err := c.sender.Resolve(chat).Join(c.ctx)
		if err != nil {
			fmt.Println("Ошибка при добавлении в чат: " + err.Error())
			fmt.Println(chat)
			time.Sleep(time.Second * 30)
			continue
		}
		c.addedChats = append(c.addedChats, chat)
		time.Sleep(time.Second * 15)
	}
	currentList := c.config.Phone[c.phone[1:]].AddedChats
	newAddedChatList := append(currentList, c.addedChats...)
	c.config.Phone[c.phone[1:]].AddedChats = newAddedChatList
	config.SaveConfig(c.config)

}

func ParseProxy(proxy string) Proxy {
	parts := strings.Split(proxy, ":")
	ip := parts[0]
	port := parts[1]
	login := parts[2]
	password := parts[3]
	return Proxy{
		IP:       ip,
		Port:     port,
		Login:    login,
		Password: password,
	}
}

func sessionFolder(phone string) string {
	var out []rune
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return "phone-" + string(out)
}

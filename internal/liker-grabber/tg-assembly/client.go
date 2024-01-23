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
	"strings"
	"time"
)

const (
	sessionPath = "session"
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

func NewClient(
	ctx context.Context,
	phone, proxyParams string,
	log *logrus.Logger,
	chats []string,
	messages chan *tg.Message,
	like, parse, comment bool,
	config *config.Config,
	hash string,
	id int,
	clientName string,
	comments []string,
) (*Client, error) {
	client := &Client{
		banned:  false,
		proxy:   ParseProxy(proxyParams),
		waiter:  &floodwait.Waiter{},
		phone:   phone,
		log:     log,
		appID:   id,
		appHash: hash,
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
	dispatcher.OnNewChannelMessage(OnNewChannelMessageHandler(client, like, parse, comment, clientName, comments))

	codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
		log.Print("Enter code for " + phone + ": ")
		code, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(code), nil
	}
	password := "123456789password"
	flow := auth.NewFlow(
		auth.Constant(phone, password, auth.CodeAuthenticatorFunc(codePrompt)),
		auth.SendCodeOptions{},
	)
	client.flow = flow
	return client, nil

}

func (c *Client) StartWaiter(ctx context.Context, firstName, lastName, about string) error {
	err := c.waiter.Run(c.ctx, func(ctx context.Context) error {
		// Spawning main goroutine.
		if err := c.app.Run(ctx, func(ctx context.Context) error {
			c.app.Auth()
			self, err := c.app.Self(ctx)
			if err != nil {
				return errors.Wrap(err, "call self")
			}
			name := self.FirstName
			if self.Username != "" {
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}
			fmt.Println("Current user:", name)
			c.lg.Info("Login",
				zap.String("first_name", self.FirstName),
				zap.String("last_name", self.LastName),
				zap.String("username", self.Username),
				zap.Int64("id", self.ID),
			)
			// Updating user info.
			if err := updateUserInfo(ctx, c.api, self, firstName, lastName, about); err != nil {
				log.Println("Ошибка при обновлении информации о пользователе: " + err.Error())
			}
			c.log.Println("Joining chats")
			c.JoinChats()
			c.log.Println("Listening for updates. Interrupt (Ctrl+C) to stop.")
			return c.updatesRecovery.Run(ctx, c.api, self.ID, updates.AuthOptions{
				IsBot: self.Bot,
				OnStart: func(ctx context.Context) {
					c.log.Println("Update recovery initialized and started, listening for events")
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
			c.log.Println("Ошибка при добавлении в чат: " + err.Error())
			c.log.Println(chat)
			if strings.Contains(err.Error(), "not found") {
				c.log.Println("Вероятно, пользователь заблокирован в чате, либо ссылка на чат указана неверно. Ждем 30 секунд и продолжаем.")
			}
			time.Sleep(time.Second * 30)
			continue
		}
		c.addedChats = append(c.addedChats, chat)
		time.Sleep(time.Second * 15)
	}
	//currentList := c.config.Phone[c.phone[1:]].AddedChats
	//newAddedChatList := append(currentList, c.addedChats...)
	c.config.Phone[c.phone[1:]].AddedChats = c.addedChats
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

func updateUserInfo(ctx context.Context, api *tg.Client, self *tg.User, name, lastname, about string) error {
	inputUser := &tg.InputUserSelf{}
	fullUser, err := api.UsersGetFullUser(ctx, inputUser)
	if err != nil {
		return errors.Wrap(err, "get full user")
	}
	if self.FirstName == name && self.LastName == lastname && fullUser.FullUser.About == about {
		return nil // Nothing to update.
	}
	_, err = api.AccountUpdateProfile(ctx, &tg.AccountUpdateProfileRequest{
		FirstName: name,
		LastName:  lastname,
		About:     about,
	})
	return err
}

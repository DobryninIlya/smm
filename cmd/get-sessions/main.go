package main

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	pebbledb "github.com/cockroachdb/pebble"
	figure "github.com/common-nighthawk/go-figure"
	"github.com/go-faster/errors"
	boltstor "github.com/gotd/contrib/bbolt"
	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
	"github.com/gotd/contrib/pebble"
	"github.com/gotd/contrib/storage"
	"github.com/gotd/td/telegram/dcs"
	"go.etcd.io/bbolt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/proxy"
	"golang.org/x/time/rate"
	lj "gopkg.in/natefinch/lumberjack.v2"
	"log"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"smm_media/internal/liker-grabber/config"
	tg_assembly "smm_media/internal/liker-grabber/tg-assembly"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	"github.com/gotd/td/tg"
)

var (
	MainConfigPath = path.Join("configs", "main.toml")
)

func sessionFolder(phone string) string {
	var out []rune
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return "phone-" + string(out)
}

func checkPhone(phone string) bool {
	if len(phone) > 15 {
		return false
	}

	for _, r := range phone {
		if (r < '0' || r > '9') && r != '+' {
			return false
		}
	}
	return true
}

func run(ctx context.Context, mainConfig *config.MainConfig, proxyURL string) error {

	// TG_PHONE is phone number in international format.
	// Like +4123456789.
	var phone string
	for phone == "" {
		fmt.Println("Введите номер телефона в международном формате (например, +712345678):")
		fmt.Scan(&phone)
		if phone == "" || !checkPhone(phone) {
			return errors.New("Номер телефона неверный. Попробуйте еще раз.")
			phone = ""
		}
	}
	// APP_HASH, APP_ID is from https://my.telegram.org/.
	appID := mainConfig.ApiID
	appHash := mainConfig.ApiHash
	if appHash == "" {
		return errors.New("no app hash")
	}

	// Setting up session storage.
	// This is needed to reuse session and not login every time.
	sessionDir := filepath.Join("session", sessionFolder(phone))
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return err
	}
	logFilePath := filepath.Join(sessionDir, "log.jsonl")

	fmt.Printf("Storing session in %s, logs in %s\n", sessionDir, logFilePath)

	// Setting up logging to file with rotation.
	//
	// Log to file, so we don't interfere with prompts and messages to user.
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

	sessionStorage := &telegram.FileSessionStorage{
		Path: filepath.Join(sessionDir, "session.json"),
	}
	db, err := pebbledb.Open(filepath.Join(sessionDir, "peers.pebble.db"), &pebbledb.Options{})
	if err != nil {
		return errors.Wrap(err, "create pebble storage")
	}
	defer db.Close()
	peerDB := pebble.NewPeerStorage(db)
	lg.Info("Storage", zap.String("path", sessionDir))

	dispatcher := tg.NewUpdateDispatcher()

	updateHandler := storage.UpdateHook(dispatcher, peerDB)

	boltdb, err := bbolt.Open(filepath.Join(sessionDir, "updates.bolt.db"), 0666, nil)
	if err != nil {
		return errors.Wrap(err, "create bolt storage")
	}
	updatesRecovery := updates.New(updates.Config{
		Handler: updateHandler, // using previous handler with peerDB
		Logger:  lg.Named("updates.recovery"),
		Storage: boltstor.NewStateStorage(boltdb),
	})

	// Handler of FLOOD_WAIT that will automatically retry request.
	waiter := floodwait.NewWaiter().WithCallback(func(ctx context.Context, wait floodwait.FloodWait) {
		lg.Warn("Flood wait", zap.Duration("wait", wait.Duration))
		fmt.Println("Got FLOOD_WAIT. Will retry after", wait.Duration)
	})

	// Filling client options.

	//pr := tg_assembly.ParseProxy("93.190.141.105:11799:9045007-all-country-PH:2e4f9iq3rd")
	pr := tg_assembly.ParseProxy(proxyURL)
	sock5, _ := proxy.SOCKS5("tcp", pr.IP+":"+pr.Port, &proxy.Auth{
		User:     pr.Login,
		Password: pr.Password,
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
			ratelimit.New(rate.Every(time.Millisecond*100), 5),
		},
		Resolver: dcs.Plain(dcs.PlainOptions{
			Dial: dc.DialContext,
		}),
	}
	client := telegram.NewClient(appID, appHash, options)
	//api := client.API()

	codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
		fmt.Print("Enter code: ")
		var code string
		fmt.Scan(&code)
		_, err := strconv.Atoi(code)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(code), nil
	}
	password := "123456789"
	flow := auth.NewFlow(
		auth.Constant(phone, password, auth.CodeAuthenticatorFunc(codePrompt)),
		auth.SendCodeOptions{},
	)

	// Authentication flow handles authentication process, like prompting for code and 2FA password.
	//flow := auth.NewFlow(tg_assembly.Terminal{PhoneNumber: phone}, auth.SendCodeOptions{})

	return waiter.Run(ctx, func(ctx context.Context) error {
		// Spawning main goroutine.
		if err := client.Run(ctx, func(ctx context.Context) error {
			// Perform auth if no session is available.
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				return errors.Wrap(err, "auth")
			}

			// Getting info about current user.
			self, err := client.Self(ctx)
			if err != nil {
				return errors.Wrap(err, "call self")
			}

			name := self.FirstName
			if self.Username != "" {
				// Username is optional.
				name = fmt.Sprintf("%s (@%s)", name, self.Username)
			}
			log.Println("Current user:", name)

			lg.Info("Login",
				zap.String("first_name", self.FirstName),
				zap.String("last_name", self.LastName),
				zap.String("username", self.Username),
				zap.Int64("id", self.ID),
			)

			// Waiting until context is done.
			log.Println("Аккаунт успешно добавлен")

			//return errors.New("account added, closing")
			return nil
		}); err != nil {
			return errors.Wrap(err, "run")
		}
		return nil
	})
}

func main() {
	banner := figure.NewFigure("REACTIVE bot", "", true)
	banner.Print()
	config := config.NewMainConfig()
	_, err := toml.DecodeFile(MainConfigPath, config)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	var proxyURL string
	for proxyURL == "" {
		fmt.Println("ВАЖНО! Необходимо ВСЕГДА использовать прокси.")
		fmt.Println("Введите адрес прокси в формате: ip:port:login:password")
		fmt.Scan(&proxyURL)
		proxyURL = strings.TrimSpace(proxyURL)
	}
	defer cancel()
	for {
		if err := run(ctx, config, proxyURL); err != nil {
			log.Println("Произошла ошибка при добавлении сессии: ", err)

		}
		fmt.Println("Продолжить добавление сессий? (y/n)")
		var answer string
		fmt.Scan(&answer)
		if answer == "n" {
			return
		}
	}
}

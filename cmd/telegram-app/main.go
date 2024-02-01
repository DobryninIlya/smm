package main

import (
	"context"
	"flag"
	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	telegramApp "smm_media/internal/telegram_app/app"
	"syscall"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", path.Join("configs", "telegram_app.toml"), "path to config file")
}

func main() {
	flag.Parse()
	config := telegramApp.NewConfig()
	_, err := toml.DecodeFile(configPath, config)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	var server *telegramApp.App
	var srv *http.Server

	m := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("cert"), // Кэширование сертификата
		Email:      "mr.woodysimpson@gmail.com",
		HostPolicy: autocert.HostWhitelist("bookingandrent.ru", "www.bookingandrent.ru"),
	}

	go handleSignals(cancel)
	go func() {
		if server, err = telegramApp.Start(ctx, config); err != nil {
			log.Fatal(err)
			cancel()
		}
		srv = &http.Server{
			Addr:      ":https",
			Handler:   server,
			TLSConfig: m.TLSConfig(),
		}
		if err = srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
			cancel()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			if err = srv.Shutdown(ctx); err != nil {
				log.Printf("Ошибка при остановке сервера: %v", err)
				if err = server.Close(); err != nil {
					log.Printf("Ошибка при закрытии сервера: %v", err)
				}
				return
			} else {
				if err = server.Close(); err != nil {
					log.Printf("Ошибка при закрытии сервера: %v", err)
				}
				log.Println("Сервер успешно остановлен")
				return
			}
		}
	}
}

func handleSignals(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	cancel()
}

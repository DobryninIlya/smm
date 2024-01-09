package main

import (
	"context"
	"github.com/BurntSushi/toml"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
	"log"
	"path"
	"smm_media/internal/liker-grabber/app"
	"smm_media/internal/liker-grabber/config"
)

var (
	ConfigPath = path.Join("configs", "base.toml")
)

func main() {
	config := config.NewConfig(ConfigPath)
	_, err := toml.DecodeFile(ConfigPath, config)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	logger := logrus.New()
	messages := make(chan *tg.Message)
	server := app.NewApp(ctx, logger, messages, config)
	server.Start()
}

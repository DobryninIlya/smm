package main

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/common-nighthawk/go-figure"
	"github.com/gotd/td/tg"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"path"
	"path/filepath"
	"smm_media/internal/liker-grabber/app"
	"smm_media/internal/liker-grabber/config"
	"time"
)

var (
	ConfigPath     = path.Join("configs", "base.toml")
	MainConfigPath = path.Join("configs", "main.toml")
)

func main() {
	banner := figure.NewFigure("REACTIVE bot", "", true)
	banner.Print()
	fmt.Println("========================")
	fmt.Println("Support TG @dobryninilya")
	fmt.Println("========================\n\n\n\n\n\n\n")
	mainConfig := config.NewMainConfig()
	_, err := toml.DecodeFile(MainConfigPath, mainConfig)
	if err != nil {
		log.Fatal(err)
	}
	phoneConfig := config.NewConfig(ConfigPath)
	_, err = toml.DecodeFile(ConfigPath, phoneConfig)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	logger := logrus.New()
	currentTime := time.Now()
	dateTime := currentTime.Format("2006-01-02_15-04-05")

	// Создаем название файла с текущей датой и временем
	logFileName := "app_" + dateTime + ".log"
	file, err := os.OpenFile(filepath.Join("logs", logFileName), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	// Создаем хук для записи в файл
	fileHook := &FileHook{
		file: file,
		formatter: &logrus.TextFormatter{
			ForceColors:   true,
			DisableColors: true,
		},
	}

	// Добавляем FileHook в логгер
	logger.AddHook(fileHook)

	messages := make(chan *tg.Message)
	server := app.NewApp(ctx, logger, messages, phoneConfig, mainConfig)
	server.Start()
}

type FileHook struct {
	file      *os.File
	formatter logrus.Formatter
}

// Levels возвращает уровни логирования, на которых данный хук должен реагировать
func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire вызывается при каждом логировании
func (hook *FileHook) Fire(entry *logrus.Entry) error {
	// Форматируем запись лога с использованием установленного форматтера
	line, err := hook.formatter.Format(entry)
	if err != nil {
		return err
	}

	// Пишем отформатированную запись лога в файл
	_, err = hook.file.WriteString(string(line))
	return err
}

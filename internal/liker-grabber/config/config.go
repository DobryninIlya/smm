package config

import (
	"github.com/BurntSushi/toml"
	"os"
	"smm_media/internal/liker-grabber/model"
	"sync"
)

type Config struct {
	Phone map[string]*model.Phone `toml:"phone_proxies"`
	Path  string
	sync.Mutex
}

func NewConfig(path string) *Config {
	return &Config{Path: path}
}

func SaveConfig(config *Config) error {

	config.Lock()
	defer config.Unlock()

	file, err := os.Create(config.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return err
	}

	return nil
}

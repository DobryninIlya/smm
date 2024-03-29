package telegram_app

type Config struct {
	StorePath   string `toml:"store_path"`
	BindAddr    string `toml:"bind_addr"`
	DatabaseURL string `toml:"database_url"`
}

// NewConfig ...
func NewConfig() *Config {
	return &Config{
		BindAddr: ":8283",
	}
}

package config

type MainConfig struct {
	ApiHash string `toml:"Api_Hash"`
	ApiID   int    `toml:"Api_Id"`
}

func NewMainConfig() *MainConfig {
	return &MainConfig{}
}

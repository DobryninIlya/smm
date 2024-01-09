package model

type Phone struct {
	Proxy      string   `toml:"proxy"`
	Location   string   `toml:"location"`
	Like       bool     `toml:"like"`
	Parse      bool     `toml:"parse"`
	ChatLinks  []string `toml:"chat_links"`
	AddedChats []string `toml:"added_chats"`
}

package model

type Phone struct {
	Proxy      string   `toml:"proxy"`
	Location   string   `toml:"location"`
	Like       bool     `toml:"like"`
	Parse      bool     `toml:"parse"`
	Comment    bool     `toml:"comment"`
	ChatLinks  []string `toml:"chat_links"`
	AddedChats []string `toml:"added_chats"`
}

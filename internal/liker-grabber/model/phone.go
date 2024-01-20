package model

type Phone struct {
	Name       string   `toml:"first_name"`
	LastName   string   `toml:"last_name"`
	About      string   `toml:"about"`
	Proxy      string   `toml:"proxy"`
	Location   string   `toml:"location"`
	Like       bool     `toml:"like"`
	Parse      bool     `toml:"parse"`
	Comment    bool     `toml:"comment"`
	ChatLinks  []string `toml:"chat_links"`
	AddedChats []string `toml:"added_chats"`
}

package models

type GetCarQuery struct {
	LocationSlug  string `json:"city_slug"`
	Page          int    `json:"page"`
	Pickup        string `json:"pickup"`
	Drop          string `json:"drop"`
	TransportType string `json:"transport_type"`
	MinPrice      int    `json:"min_price"`
	MaxPrice      int    `json:"max_price"`
}

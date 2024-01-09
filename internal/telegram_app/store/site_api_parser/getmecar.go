package site_api_parser

import (
	"io"
	"net/http"
	"smm_media/internal/telegram_app/models"
	"strconv"
)

const (
	domain   = "https://dev2.getmecar.ru/"
	location = "locations/"
)

type GetMeCar struct {
	url string
}

func NewGetMeCar() *GetMeCar {
	return &GetMeCar{url: domain}
}

func (g *GetMeCar) GetCarsPage(query models.GetCarQuery, page int) ([]byte, error) {
	resp, err := http.Get(formGetCarsUrl(query, page))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//html, err := deleteTagsWithClasses(body, []string{})
	if err != nil {
		return nil, err
	}
	return body, nil
}

func formGetCarsUrl(query models.GetCarQuery, page int) string {
	return domain + location + query.LocationSlug + "/page/" + strconv.Itoa(page) + "?" +
		"pickup=" + query.Pickup + "&" + "drop=" + query.Drop + "&" + "transport=" + query.TransportType + "&" +
		"min_price=" + strconv.Itoa(query.MinPrice) + "&" + "max_price=" + strconv.Itoa(query.MaxPrice) + "&il=1"
}

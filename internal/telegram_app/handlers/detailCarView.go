package handlers

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"smm_media/internal/telegram_app/page_constructor"
	site "smm_media/internal/telegram_app/store/site_api_parser"
)

type LinkStruct struct {
	Link string `json:"link"`
}

func NewDetailCarPageHandler(log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const path = "handlers.api.detailCarView.NewDetailCarPageHandler"
		url := r.URL
		linkParam := url.Query().Get("link")
		fmt.Println(linkParam)
		gmc := site.NewGetMeCar()
		carPage, err := gmc.GetDetailedCarPage(linkParam)
		mainBlock := site.FormPersonalCarPage(carPage)
		if err != nil {
			log.WithFields(logrus.Fields{
				"path": path,
			}).Error(err)
			Respond(w, r, 400, "Bad request")
			return
		}
		page := page_constructor.GetDetailedCarPage(mainBlock)
		Respond(w, r, 200, page)
	}
}

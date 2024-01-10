package handlers

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"smm_media/internal/telegram_app/page_constructor"
	site "smm_media/internal/telegram_app/store/site_api_parser"
)

func NewDetailCarPageHandler(log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const path = "handlers.api.detailCarView.NewDetailCarPageHandler"
		gmc := site.NewGetMeCar()
		carPage, err := gmc.GetDetailedCarPage("https://dev2.getmecar.ru/listing/hyundai-accent-2014-2016-ili-analog-v-baku-azerbajdzhan/?location=azerbajdzhan&pickup=1704875760&drop=1705480560")
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

package handlers

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"io"
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
		var link LinkStruct
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Logf(
				logrus.ErrorLevel,
				"%s : Ошибка получение body: %v",
				path,
				err.Error(),
			)
			ErrorHandlerAPI(w, r, http.StatusBadRequest, ErrInternal)
			return
		}
		err = json.Unmarshal(body, &link)
		if err != nil {
			log.Logf(
				logrus.ErrorLevel,
				"%s : Ошибка анмаршалинга body: %v",
				path,
				err.Error(),
			)
			ErrorHandlerAPI(w, r, http.StatusBadRequest, ErrBadPayload)
			return
		}
		gmc := site.NewGetMeCar()
		carPage, err := gmc.GetDetailedCarPage(link.Link)
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

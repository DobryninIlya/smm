package handlers

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"smm_media/internal/telegram_app/models"
	site "smm_media/internal/telegram_app/store/site_api_parser"
	"strconv"
)

func NewGetCarsHandler(log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const path = "handlers.api.getCarsCards.NewGetCarsHandler"
		url := r.URL
		query := url.Query()
		page := query.Get("page")
		pickup := query.Get("pickup")
		drop := query.Get("drop")
		transport := query.Get("transport")
		minPrice := query.Get("min_price")
		maxPrice := query.Get("max_price")
		locationSlug := query.Get("location_slug")

		if locationSlug == "" || pickup == "" || drop == "" || transport == "" {
			log.WithFields(logrus.Fields{
				"path": path,
			}).Error("Bad request")
			Respond(w, r, 400, "Заполните необходимые поля")
			return
		}

		pageI, err := strconv.Atoi(page)
		if err != nil && page != "" {
			log.WithFields(logrus.Fields{
				"path": path,
				"page": page,
			}).Error(err)
			Respond(w, r, 400, "Заполните необходимые поля")
			return
		}
		if page == "" {
			pageI = 1
		}
		var minPriceI int
		if minPrice != "" {
			minPriceI, err = strconv.Atoi(minPrice)
			if err != nil {
				log.WithFields(logrus.Fields{
					"path":      path,
					"min_price": minPrice,
				}).Error(err)
				Respond(w, r, 400, "Заполните необходимые поля")
				return
			}
		} else {
			minPriceI = 0
		}
		var maxPriceI int
		if maxPrice != "" {
			maxPriceI, err = strconv.Atoi(maxPrice)
			if err != nil {
				log.WithFields(logrus.Fields{
					"path":      path,
					"max_price": maxPrice,
				}).Error(err)
				Respond(w, r, 400, "Заполните необходимые поля")
				return
			}
		} else {
			maxPriceI = 0
		}

		queryStruct := models.GetCarQuery{
			LocationSlug:  locationSlug,
			Page:          pageI,
			Pickup:        pickup,
			Drop:          drop,
			TransportType: transport,
			MinPrice:      minPriceI,
			MaxPrice:      maxPriceI,
		}
		gmc := site.NewGetMeCar()
		resultPage, err := gmc.GetCarsPage(queryStruct, pageI)
		if err != nil {
			log.WithFields(logrus.Fields{
				"path": path,
			}).Error(err)
			Respond(w, r, 500, "Internal server error")
			return
		}
		Respond(w, r, 200, resultPage)
	}
}

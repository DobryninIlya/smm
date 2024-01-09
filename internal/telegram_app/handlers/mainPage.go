package handlers

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"smm_media/internal/telegram_app/page_constructor"
)

func NewMainHandler(log *logrus.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		const path = "handlers.api.mainPage.NewMainHandler"
		page := page_constructor.GetMainPage()
		Respond(w, r, 200, page)
	}
}

package telegram_app

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"net/http"
	"path/filepath"
	handler "smm_media/internal/telegram_app/handlers"
	"smm_media/internal/telegram_app/page_constructor"
	"time"
)

type App struct {
	router *chi.Mux
	server *http.Server
	//store    sqlstore.StoreInterface
	logger   *logrus.Logger
	ctx      context.Context
	filePath string
}

func newApp(ctx context.Context, bindAddr string, config Config) *App {
	router := chi.NewRouter()
	server := &http.Server{
		Addr:    bindAddr,
		Handler: router,
	}
	logger := logrus.New()
	a := &App{
		router: router,
		server: server,
		//store:    store,
		logger:   logger,
		ctx:      ctx,
		filePath: config.StorePath,
	}
	a.configureRouter()
	return a
}

func (a *App) Close() error {
	err := a.server.Close()
	if err != nil {
		return err
	}
	return a.server.Close()
}

func (a *App) configureRouter() {
	a.router.Use(a.logRequest)
	a.router.Route("/app", func(r chi.Router) {
		r.Get("/", handler.NewMainHandler(a.logger))
		r.Get("/get_cars", handler.NewGetCarsHandler(a.logger))
		r.Get("/rent", handler.NewDetailCarPageHandler(a.logger))
	})
	a.router.Handle("/static/css/*", http.StripPrefix("/static/css/", cssHandler(http.FileServer(http.Dir(filepath.Join(page_constructor.Path, "css"))))))
	a.router.Handle("/static/js/*", http.StripPrefix("/static/js/", http.FileServer(http.Dir(filepath.Join(page_constructor.Path, "js")))))
	a.router.Handle("/static/img/*", http.StripPrefix("/static/img/", http.FileServer(http.Dir(filepath.Join(page_constructor.Path, "img")))))
	a.router.Handle("/static/json/*", http.StripPrefix("/static/json/", http.FileServer(http.Dir(filepath.Join(page_constructor.Path, "json")))))

}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

func cssHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		next.ServeHTTP(w, r)
	})
}

func (a *App) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := a.logger.WithFields(logrus.Fields{
			"remote_addr": r.RemoteAddr,
		})
		logger.Infof("started %s %s", r.Method, r.RequestURI)

		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		var level logrus.Level
		switch {
		case rw.code >= 500:
			level = logrus.ErrorLevel
		case rw.code >= 400:
			level = logrus.WarnLevel
		default:
			level = logrus.InfoLevel
		}
		logger.Logf(
			level,
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Now().Sub(start),
		)
	})
}

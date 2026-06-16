package http

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	urlHandler *URLHandler
	log        *slog.Logger
}

func NewHandler(urlHandler *URLHandler, log *slog.Logger) *Handler {
	return &Handler{
		urlHandler: urlHandler,
		log:        log,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(h.loggerMiddleware)

	// Редирект — главный эндпоинт, на корне роутера
	r.Get("/{code}", h.urlHandler.Redirect)

	// REST API
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", h.health)

		r.Route("/urls", func(r chi.Router) {
			r.Post("/", h.urlHandler.Create)
			r.Delete("/{code}", h.urlHandler.Delete)
			r.Get("/", h.urlHandler.List)
		})
	})

	return r
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		h.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/cageymage/fuzion/backend/internal/news"
	"github.com/cageymage/fuzion/backend/internal/raids"
	"github.com/cageymage/fuzion/backend/internal/streams"
)

type Deps struct {
	News           *news.Handler
	Raids          *raids.Handler
	Streams        *streams.Handler
	AllowedOrigins []string
}

func New(deps Deps) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Credentials are on so the session cookie survives the cross-subdomain
	// hop between app.fuzion.gg and api.fuzion.gg.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   deps.AllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", health)
		deps.News.Register(api)
		deps.Raids.Register(api)
		deps.Streams.Register(api)
	})

	return r
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

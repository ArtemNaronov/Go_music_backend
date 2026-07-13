package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/temic/go-music/internal/auth"
)

// NewRouter builds the HTTP router with middleware and routes.
func NewRouter(token string, handler *Handler, logger zerolog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health)

	r.Route("/api", func(r chi.Router) {
		r.Use(auth.BearerToken(token))

		r.Group(func(r chi.Router) {
			r.Use(requestLogger(logger))

			r.Get("/artists", handler.ListArtists)
			r.Get("/albums", handler.ListAlbums)
			r.Get("/search", handler.Search)
			r.Get("/stations", handler.ListStations)
			r.Get("/albums/{id}/tracks", handler.ListAlbumTracks)
			r.Post("/rescan", handler.Rescan)
			r.Get("/radio", handler.Radio)

			r.Get("/tracks", handler.ListTracks)
			r.Get("/tracks/{id}", handler.GetTrack)
		})

		r.Get("/albums/{id}/cover", handler.CoverAlbum)
		r.Get("/tracks/{id}/cover", handler.CoverTrack)
		r.Get("/tracks/{id}/stream", handler.StreamTrack)
		r.Head("/tracks/{id}/stream", handler.StreamTrack)
	})

	return r
}

func requestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info().
					Str("method", r.Method).
					Str("path", r.URL.Path).
					Int("status", ww.Status()).
					Dur("duration", time.Since(started)).
					Msg("request")
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

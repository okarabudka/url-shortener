// Package remove serves DELETE /url/{alias}: it deletes the link stored under the alias.
package remove

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "github.com/okarabudka/url-shortener/internal/lib/api/response"
	"github.com/okarabudka/url-shortener/internal/lib/logger/sl"
	"github.com/okarabudka/url-shortener/internal/storage"
)

// URLDeleter deletes the link stored under an alias.
type URLDeleter interface {
	DeleteURL(alias string) error
}

// New returns the delete handler. The route must define the {alias} URL parameter.
func New(log *slog.Logger, urlDeleter URLDeleter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.remove.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")

		err := urlDeleter.DeleteURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", slog.String("alias", alias))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("not found"))

			return
		}
		if err != nil {
			log.Error("failed to delete url", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))

			return
		}

		log.Info("url deleted", slog.String("alias", alias))

		w.WriteHeader(http.StatusNoContent)
	}
}

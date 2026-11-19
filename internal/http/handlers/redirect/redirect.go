// Package redirect serves GET /{alias}: it redirects to the URL stored under the alias.
package redirect

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

// URLGetter looks up the URL stored under an alias.
type URLGetter interface {
	GetURL(alias string) (string, error)
}

// New returns the redirect handler. The route must define the {alias} URL parameter.
func New(log *slog.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")

		target, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", slog.String("alias", alias))

			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.Error("not found"))

			return
		}
		if err != nil {
			log.Error("failed to get url", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("internal error"))

			return
		}

		log.Info("redirecting", slog.String("alias", alias), slog.String("url", target))

		//nolint:gosec // Redirecting to stored URLs is the purpose; only authenticated users store them, and only http(s).
		http.Redirect(w, r, target, http.StatusFound)
	}
}

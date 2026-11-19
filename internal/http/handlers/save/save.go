// Package save serves POST /url: it stores a URL under a given or generated alias.
package save

import (
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	resp "github.com/okarabudka/url-shortener/internal/lib/api/response"
	"github.com/okarabudka/url-shortener/internal/lib/logger/sl"
	"github.com/okarabudka/url-shortener/internal/lib/random"
	"github.com/okarabudka/url-shortener/internal/storage"
)

// Request is the body of POST /url.
type Request struct {
	// http_url only accepts http and https, so links cannot point to javascript: or file: URLs.
	URL   string `json:"url" validate:"required,http_url"`
	Alias string `json:"alias,omitempty" validate:"omitempty,alphanum,max=32"`
}

// Response is the body of a successful POST /url.
type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

// URLSaver stores a URL under an alias.
type URLSaver interface {
	SaveURL(urlToSave string, alias string) (int64, error)
}

// New returns the save handler. Aliases generated for requests without one have aliasLength characters.
func New(log *slog.Logger, urlSaver URLSaver, aliasLength int) http.HandlerFunc {
	validate := validator.New()

	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.save.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request

		err := render.DecodeJSON(r.Body, &req)
		if errors.Is(err, io.EOF) {
			log.Error("request body is empty")

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("empty request"))

			return
		}
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.Error("failed to decode request"))

			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validate.Struct(req); err != nil {
			var validateErr validator.ValidationErrors
			errors.As(err, &validateErr)

			log.Error("invalid request", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		alias := req.Alias
		if alias == "" {
			alias = random.NewRandomString(aliasLength)
		}

		id, err := urlSaver.SaveURL(req.URL, alias)
		if errors.Is(err, storage.ErrURLExists) {
			log.Info("alias already exists", slog.String("alias", alias))

			render.Status(r, http.StatusConflict)
			render.JSON(w, r, resp.Error("alias already exists"))

			return
		}
		if err != nil {
			log.Error("failed to add url", sl.Err(err))

			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.Error("failed to add url"))

			return
		}

		log.Info("url added", slog.Int64("id", id))

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Alias:    alias,
		})
	}
}

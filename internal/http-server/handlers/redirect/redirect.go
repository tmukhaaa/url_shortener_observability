package redirect

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"golang.org/x/exp/slog"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/lib/metrics"
	"url-shortener/internal/storage"
)

// URLGetter is an interface for getting url by alias.
//
//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (string, error)
}

// handler holds dependencies for the redirect handler
type handler struct {
	log       *slog.Logger
	urlGetter URLGetter
	metrics   *metrics.Metrics
}

// New creates a new redirect handler
func New(log *slog.Logger, urlGetter URLGetter, metrics *metrics.Metrics) http.HandlerFunc {
	h := &handler{
		log:       log,
		urlGetter: urlGetter,
		metrics:   metrics,
	}

	return h.Handle
}

// Handle processes the redirect request
func (h *handler) Handle(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.url.redirect.Handle"

	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	alias := chi.URLParam(r, "alias")
	if alias == "" {
		log.Info("alias is empty")

		render.JSON(w, r, resp.Error("invalid request"))

		return
	}

	resURL, err := h.urlGetter.GetURL(alias)
	if errors.Is(err, storage.ErrURLNotFound) {
		log.Info("url not found", "alias", alias)

		render.JSON(w, r, resp.Error("not found"))

		return
	}
	if err != nil {
		log.Error("failed to get url", sl.Err(err))

		render.JSON(w, r, resp.Error("internal error"))

		return
	}

	log.Info("got url", slog.String("url", resURL))

	// Record the redirect in metrics
	h.metrics.RecordRedirect()

	// redirect to found url
	http.Redirect(w, r, resURL, http.StatusFound)
}

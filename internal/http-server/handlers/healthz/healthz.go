package healthz

import (
	"net/http"

	"github.com/go-chi/render"
)

// HealthzResponse is the response for the health check endpoint
type HealthzResponse struct {
	Status string `json:"status"`
}

// Healthz handles the health check endpoint
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, HealthzResponse{
			Status: "ok",
		})
	}
}

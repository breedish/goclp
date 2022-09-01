package handlers

import (
	"canvas/views"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func FrontPage(mux chi.Router) {
	mux.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_ = views.FrontPage().Render(w)
	})
}

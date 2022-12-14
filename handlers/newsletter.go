package handlers

import (
	"canvas/model"
	"canvas/views"
	"context"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
)

type signupper interface {
	SignupForNewsletter(ctx context.Context, email model.Email) (string, error)
}

type sender interface {
	Send(ctx context.Context, m model.Message) error
}

func NewsletterSignup(mux chi.Router, s signupper, q sender, log *zap.Logger) {
	mux.Post("/newsletter/signup", func(w http.ResponseWriter, r *http.Request) {
		email := model.Email(r.FormValue("email"))

		if !email.IsValid() {
			http.Error(w, "email is invalid", http.StatusBadRequest)
			return
		}

		token, err := s.SignupForNewsletter(r.Context(), email)
		if err != nil {
			log.Info("Error signing up for newsletter", zap.Error(err))
			http.Error(w, "error signing up, refresh to try again", http.StatusBadGateway)
			return
		}

		err = q.Send(r.Context(), model.Message{
			"job":   "confirmation_email",
			"email": email.String(),
			"token": token,
		})
		if err != nil {
			http.Error(w, "error signing up, refresh to try again", http.StatusBadGateway)
			return
		}

		http.Redirect(w, r, "/newsletter/thanks", http.StatusFound)
	})
}

func NewsletterThanks(mux chi.Router) {
	mux.Get("/newsletter/thanks", func(w http.ResponseWriter, r *http.Request) {
		_ = views.NewsletterThanksPage("/newsletter/thanks").Render(w)
	})
}

type confirmer interface {
	ConfirmNewsletterSignup(ctx context.Context, token string) (*model.Email, error)
}

func NewsletterConfirm(mux chi.Router, s confirmer, q sender) {
	mux.Get("/newsletter/confirm", func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")

		_ = views.NewsletterConfirmPage("/newsletter/confirm", token).Render(w)
	})

	mux.Post("/newsletter/confirm", func(w http.ResponseWriter, r *http.Request) {
		token := r.FormValue("token")

		email, err := s.ConfirmNewsletterSignup(r.Context(), token)
		if err != nil {
			http.Error(w, "error saving email address confirmation, refresh to try again", http.StatusBadGateway)
			return
		}
		if email == nil {
			http.Error(w, "bad token", http.StatusBadRequest)
			return
		}

		err = q.Send(r.Context(), model.Message{
			"job":   "welcome_email",
			"email": email.String(),
		})
		if err != nil {
			http.Error(w, "error saving email address confirmation, refresh to try again", http.StatusBadGateway)
			return
		}

		http.Redirect(w, r, "/newsletter/confirmed", http.StatusFound)
	})
}

func NewsletterConfirmed(mux chi.Router) {
	mux.Get("/newsletter/confirmed", func(w http.ResponseWriter, r *http.Request) {
		_ = views.NewsletterConfirmedPage("/newsletter/confirmed").Render(w)
	})
}

type getter interface {
	GetNewsletter(ctx context.Context, id string) (*model.Newsletter, error)
	GetNewsletters(ctx context.Context) ([]model.Newsletter, error)
}

func Newsletters(mux chi.Router, g getter, log *zap.Logger) {
	mux.Get("/newsletters", func(w http.ResponseWriter, r *http.Request) {
		id := model.UUID(r.URL.Query().Get("id"))
		if id != "" {
			if !id.IsValid() {
				http.NotFound(w, r)
				return
			}

			n, err := g.GetNewsletter(r.Context(), id.String())
			if err != nil {
				log.Info("Error getting newsletter", zap.Error(err), zap.Stringer("id", id))
				http.Error(w, "error getting newsletter, refresh to try again", http.StatusBadGateway)
				return
			}
			if n == nil {
				http.NotFound(w, r)
				return
			}
			_ = views.NewsletterPage("/newsletters", *n).Render(w)
			return
		}

		newsletters, err := g.GetNewsletters(r.Context())
		if err != nil {
			log.Info("Error getting newsletters", zap.Error(err))
			http.Error(w, "error getting newsletters, refresh to try again", http.StatusBadGateway)
			return
		}
		_ = views.NewslettersPage("/newsletters", newsletters).Render(w)
	})
}

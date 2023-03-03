package seeder

import "github.com/go-chi/chi/v5"

func secureHandler() *chi.Mux {
	return chi.NewRouter()
}

package seeder

import "github.com/go-chi/chi/v5"

func insecureHandler() *chi.Mux {
	return chi.NewRouter()
}

package seeder

import "github.com/go-chi/chi/v5"

func (s *seeder) secureHandler() *chi.Mux {
	return chi.NewRouter()
}

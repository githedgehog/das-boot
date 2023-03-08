package seeder

import "github.com/go-chi/chi/v5"

const (
	stage1PathBase = "/stage1/"
)

func (s *seeder) secureHandler() *chi.Mux {
	return chi.NewRouter()
}

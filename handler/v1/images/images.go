package images

import (
	"net/http"

	"github.com/imager/model"
)

type Service struct {
	repo model.ImagesRepository
}

func NewService(repo model.ImagesRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) All(w http.ResponseWriter, r *http.Request) {}

func (s *Service) Resize(w http.ResponseWriter, r *http.Request) {}

func (s *Service) OnlyResized(w http.ResponseWriter, r *http.Request) {}

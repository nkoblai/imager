package images

import (
	"bytes"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/imager/model"
	"github.com/imager/uploader"

	"github.com/disintegration/imaging"
)

type Service struct {
	repo     model.ImagesRepository
	uploader uploader.Service
}

func NewService(repo model.ImagesRepository, uploader uploader.Service) *Service {
	return &Service{repo: repo, uploader: uploader}
}

func (s *Service) All(w http.ResponseWriter, r *http.Request) {}

func (s *Service) Resize(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func(w http.ResponseWriter, r *http.Request) ([]byte, int) {
		file, h, err := r.FormFile("file")
		if err != nil {
			return []byte(fmt.Sprintf("error decoding file into image: %v", err)),
				http.StatusBadRequest
		}
		img, err := imaging.Decode(file)
		if err != nil {
			return []byte(fmt.Sprintf("error decoding file into image: %v", err)),
				http.StatusInternalServerError
		}

		// TODO: resize image using query vars
		img = imaging.Resize(img, 128, 128, imaging.NearestNeighbor)

		buf := new(bytes.Buffer)
		if err := imaging.Encode(buf, img, imaging.PNG); err != nil {
			return []byte(fmt.Sprintf("error encoding file to buffer: %v", err)),
				http.StatusInternalServerError
		}
		downloadURL, err := s.uploader.Upload(h.Filename, buf)
		if err != nil {
			return []byte(fmt.Sprintf("error uploading image: %v", err)),
				http.StatusInternalServerError
		}

		// TODO: store image inside DB with provided URL
		_ = downloadURL

		// TODO: provide this variable with appropriate values
		res := model.OriginalResized{
			Original: model.Image{},
			Resized:  model.Image{},
		}

		b, err := json.Marshal(res)
		if err != nil {
			return []byte(fmt.Sprintf("error marshaling result: %v", err)),
				http.StatusInternalServerError
		}
		return b, http.StatusCreated
	}(w, r)
	w.Header().Add("Conent-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}

func (s *Service) OnlyResized(w http.ResponseWriter, r *http.Request) {}

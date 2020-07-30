package images

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

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

func (s *Service) All(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func() ([]byte, int) {
		images, err := s.repo.All()
		if err != nil {
			return []byte(fmt.Sprintf("error getting images from db: %v", err)),
				http.StatusInternalServerError
		}
		res, err := json.Marshal(images)
		if err != nil {
			return []byte(fmt.Sprintf("error during marshaling images: %v", err)),
				http.StatusInternalServerError
		}
		return res, http.StatusOK
	}()
	response(w, data, statusCode)
}

func (s *Service) Resize(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func(w http.ResponseWriter, r *http.Request) ([]byte, int) {
		// TODO: use context recieved from request for futher func calls

		file, h, err := r.FormFile("file")
		if err != nil {
			return []byte(fmt.Sprintf("error decoding file %s into image: %v", h.Filename, err)),
				http.StatusBadRequest
		}
		defer file.Close()

		oldImgBytes, err := ioutil.ReadAll(file)
		if err != nil {
			return []byte(fmt.Sprintf("error reading file %s with error: %v", h.Filename, err)),
				http.StatusBadRequest
		}

		img, err := imaging.Decode(bytes.NewReader(oldImgBytes))
		if err != nil {
			return []byte(fmt.Sprintf("error decoding file %s into image: %v", h.Filename, err)),
				http.StatusInternalServerError
		}

		originalImageResolution := fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy())

		// TODO: resize image using query vars
		img = imaging.Resize(img, 128, 128, imaging.NearestNeighbor)
		imgFormat := imaging.PNG

		buf := new(bytes.Buffer)
		if err := imaging.Encode(buf, img, imgFormat); err != nil {
			return []byte(fmt.Sprintf("error encoding file %s to buffer: %v", h.Filename, err)),
				http.StatusInternalServerError
		}

		newImgBytes := buf.Bytes()

		res, err := s.uploadImages([2][]byte{oldImgBytes, newImgBytes}, imgFormat.String())
		if err != nil {
			return []byte(fmt.Sprintf("error uploading images: %v", err)),
				http.StatusInternalServerError
		}

		res.Original.Resolution = originalImageResolution
		// TODO: use query vars
		res.Resized.Resolution = fmt.Sprintf("%dx%d", 128, 128)

		originalID, err := s.repo.Save(res.Original)
		if err != nil {
			return []byte(err.Error()),
				http.StatusInternalServerError
		}

		res.Original.ID = originalID
		res.Resized.OriginalID = originalID

		resizedID, err := s.repo.Save(res.Resized)
		if err != nil {
			return []byte(err.Error()),
				http.StatusInternalServerError
		}

		res.Resized.ID = resizedID

		b, err := json.Marshal(res)
		if err != nil {
			return []byte(fmt.Sprintf("error marshaling result: %v", err)),
				http.StatusInternalServerError
		}
		return b, http.StatusCreated
	}(w, r)
	response(w, data, statusCode)
}

func calcualteMD5(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		if err != nil {
			return "", fmt.Errorf("calculating md5 was failed with error: %v", err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func name(hash string, format string) string {
	return fmt.Sprintf("%s.%s", hash, strings.ToLower(format))
}

func (s *Service) uploadImages(images [2][]byte, format string) (model.OriginalResized, error) {
	var res model.OriginalResized
	wg := sync.WaitGroup{}
	wg.Add(len(images))
	errCh := make(chan error, len(images))

	go func() {
		defer wg.Done()
		hash, err := calcualteMD5(bytes.NewBuffer(images[0]))
		if err != nil {
			errCh <- err
			return
		}
		res.Original.DownloadURL, err = s.uploader.Upload(
			name(hash, format),
			bytes.NewBuffer(images[0]),
		)
		errCh <- err
	}()

	go func() {
		defer wg.Done()
		hash, err := calcualteMD5(bytes.NewBuffer(images[1]))
		if err != nil {
			errCh <- err
			return
		}
		res.Resized.DownloadURL, err = s.uploader.Upload(
			name(hash, format),
			bytes.NewBuffer(images[1]),
		)
		errCh <- err
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return model.OriginalResized{}, err
		}
	}

	return res, nil
}

func (s *Service) OnlyResized(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func() ([]byte, int) {
		images, err := s.repo.OnlyResized()
		if err != nil {
			return []byte(fmt.Sprintf("error getting resized images from db: %v", err)),
				http.StatusInternalServerError
		}
		res, err := json.Marshal(images)
		if err != nil {
			return []byte(fmt.Sprintf("error during marshaling images: %v", err)),
				http.StatusInternalServerError
		}
		return res, http.StatusOK
	}()
	response(w, data, statusCode)
}

func response(w http.ResponseWriter, data []byte, statusCode int) {
	w.Header().Add("Conent-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}

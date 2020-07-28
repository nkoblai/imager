package images

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
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

func (s *Service) All(w http.ResponseWriter, r *http.Request) {}

func (s *Service) Resize(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func(w http.ResponseWriter, r *http.Request) ([]byte, int) {
		file, h, err := r.FormFile("file")
		if err != nil {
			return []byte(fmt.Sprintf("error decoding file into image: %v", err)),
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

		// TODO: resize image using query vars
		img = imaging.Resize(img, 128, 128, imaging.NearestNeighbor)
		imgFormat := imaging.PNG

		buf := new(bytes.Buffer)
		if err := imaging.Encode(buf, img, imgFormat); err != nil {
			return []byte(fmt.Sprintf("error encoding file to buffer: %v", err)),
				http.StatusInternalServerError
		}

		newImgBytes := buf.Bytes()

		res, err := s.uploadImages([2][]byte{oldImgBytes, newImgBytes})
		if err != nil {
			return []byte(fmt.Sprintf("error uploading images: %v", err)),
				http.StatusInternalServerError
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

func calcualteMD5(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		if err != nil {
			return "", fmt.Errorf("calculating md5 was failed with error: %v", err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (s *Service) uploadImages(images [2][]byte) (model.OriginalResized, error) {
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
			fmt.Sprintf("%s.%s", hash, strings.ToLower(imaging.PNG.String())),
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
			fmt.Sprintf("%s.%s", hash, strings.ToLower(imaging.PNG.String())),
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

func (s *Service) OnlyResized(w http.ResponseWriter, r *http.Request) {}

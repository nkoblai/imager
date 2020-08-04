package handler

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/imager/model"
	"github.com/imager/web/downloader"
	"github.com/imager/web/uploader"

	"github.com/disintegration/imaging"
)

const imgFormat = imaging.PNG

// Service represents handler service.
type Service struct {
	repo       model.ImagesRepository
	uploader   uploader.Service
	downloader downloader.Service
}

// NewService returns new handler service.
func NewService(repo model.ImagesRepository, uploader uploader.Service, downloader downloader.Service) *Service {
	return &Service{repo: repo, uploader: uploader, downloader: downloader}
}

// All returns all images.
func (s *Service) All(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func() ([]byte, int) {
		ctx := r.Context()
		images, err := s.repo.All(ctx)
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

// ResizeByID uses for changing existing image.
func (s *Service) ResizeByID(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func(w http.ResponseWriter, r *http.Request) ([]byte, int) {
		ctx := r.Context()
		weight, height, err := validateSizeParams(r)
		if err != nil {
			return []byte(fmt.Sprintf("error validating resize params: %v", err)),
				http.StatusBadRequest
		}
		id, err := strconv.Atoi(mux.Vars(r)["id"])
		if err != nil {
			return []byte(fmt.Sprintf("error converting id to int: %v", err)),
				http.StatusBadRequest
		}
		originalImage, err := s.repo.GetOne(ctx, id)
		if err != nil {
			return []byte(fmt.Sprintf("couldn't get image by id: %d with error: %v", id, err)),
				http.StatusInternalServerError
		}

		originalImageName := path.Base(originalImage.DownloadURL)

		oldImgBytes, err := s.downloader.Download(ctx, originalImage.DownloadURL)
		if err != nil {
			return []byte(fmt.Sprintf("couldn't download image by url: %s with error: %v", originalImageName, err)),
				http.StatusInternalServerError
		}

		newImageResolution := fmt.Sprintf("%dx%d", weight, height)

		img, err := imaging.Decode(bytes.NewReader(oldImgBytes))
		if err != nil {
			return []byte(fmt.Sprintf("error decoding file %s into image: %v", originalImageName, err)),
				http.StatusInternalServerError
		}

		img = imaging.Resize(img, weight, height, imaging.NearestNeighbor)
		if img == nil {
			return []byte(fmt.Sprintf("couldn't resize image '%s'", originalImageName)),
				http.StatusInternalServerError
		}

		buf := new(bytes.Buffer)
		if err := imaging.Encode(buf, img, imgFormat); err != nil {
			return []byte(fmt.Sprintf("error encoding file %s to buffer: %v", originalImageName, err)),
				http.StatusInternalServerError
		}

		hash, err := calculateMD5(bytes.NewBuffer(buf.Bytes()))
		if err != nil {
			return []byte(fmt.Sprintf("error calculating md5 for image %v", err)),
				http.StatusInternalServerError
		}

		downloadURL, err := s.uploader.Upload(ctx, name(hash), buf)
		if err != nil {
			return []byte(fmt.Sprintf("error downloading image %v", err)),
				http.StatusInternalServerError
		}

		newImage := model.Image{
			DownloadURL: downloadURL,
			Resolution:  newImageResolution,
			OriginalID:  originalImage.ID,
		}

		id, err = s.repo.Save(ctx, newImage)
		if err != nil {
			return []byte(err.Error()),
				http.StatusInternalServerError
		}
		newImage.ID = id

		res := model.OriginalResized{
			Original: originalImage,
			Resized:  newImage,
		}

		b, err := json.Marshal(res)
		if err != nil {
			return []byte(fmt.Sprintf("error marshaling result: %v", err)),
				http.StatusInternalServerError
		}
		return b, http.StatusCreated
	}(w, r)
	response(w, data, statusCode)
}

// Resize creates and resizes image.
func (s *Service) Resize(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func(w http.ResponseWriter, r *http.Request) ([]byte, int) {
		ctx := r.Context()
		weight, height, err := validateSizeParams(r)
		if err != nil {
			return []byte(fmt.Sprintf("error validating resize params: %v", err)),
				http.StatusBadRequest
		}

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
		img = imaging.Resize(img, weight, height, imaging.NearestNeighbor)
		if img == nil {
			return []byte(fmt.Sprintf("couldn't resize image '%s'", h.Filename)),
				http.StatusInternalServerError
		}

		buf := new(bytes.Buffer)
		if err := imaging.Encode(buf, img, imgFormat); err != nil {
			return []byte(fmt.Sprintf("error encoding file %s to buffer: %v", h.Filename, err)),
				http.StatusInternalServerError
		}

		newImgBytes := buf.Bytes()

		res, err := s.uploadImages(ctx, [2][]byte{oldImgBytes, newImgBytes})
		if err != nil {
			return []byte(fmt.Sprintf("error uploading images: %v", err)),
				http.StatusInternalServerError
		}

		res.Original.Resolution = originalImageResolution
		res.Resized.Resolution = fmt.Sprintf("%dx%d", weight, height)

		originalID, err := s.repo.Save(ctx, res.Original)
		if err != nil {
			return []byte(err.Error()),
				http.StatusInternalServerError
		}

		res.Original.ID = originalID
		res.Resized.OriginalID = originalID

		resizedID, err := s.repo.Save(ctx, res.Resized)
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

func calculateMD5(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		if err != nil {
			return "", fmt.Errorf("calculating md5 was failed with error: %v", err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func name(hash string) string {
	return fmt.Sprintf("%s.%s", hash, strings.ToLower(imgFormat.String()))
}

func (s *Service) uploadImages(ctx context.Context, images [2][]byte) (model.OriginalResized, error) {
	var res model.OriginalResized
	wg := sync.WaitGroup{}
	wg.Add(len(images))
	errCh := make(chan error, len(images))

	go func() {
		defer wg.Done()
		hash, err := calculateMD5(bytes.NewBuffer(images[0]))
		if err != nil {
			errCh <- err
			return
		}
		res.Original.DownloadURL, err = s.uploader.Upload(
			ctx,
			name(hash),
			bytes.NewBuffer(images[0]),
		)
		errCh <- err
	}()

	go func() {
		defer wg.Done()
		hash, err := calculateMD5(bytes.NewBuffer(images[1]))
		if err != nil {
			errCh <- err
			return
		}
		res.Resized.DownloadURL, err = s.uploader.Upload(
			ctx,
			name(hash),
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

// OnlyResized returns only resized images.
func (s *Service) OnlyResized(w http.ResponseWriter, r *http.Request) {
	data, statusCode := func() ([]byte, int) {
		ctx := r.Context()
		images, err := s.repo.OnlyResized(ctx)
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

func validateSizeParams(r *http.Request) (int, int, error) {
	w, err := strconv.Atoi(r.URL.Query().Get("weight"))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid weight param")
	}
	h, err := strconv.Atoi(r.URL.Query().Get("height"))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height param")
	}
	if w <= 0 || w > 3840 {
		return 0, 0, fmt.Errorf("weight is not in range [0-3840]")
	}
	if h <= 0 || w > 2160 {
		return 0, 0, fmt.Errorf("height is not in range [0-2160]")
	}
	return w, h, nil
}

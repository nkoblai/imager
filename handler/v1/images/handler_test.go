package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/disintegration/imaging"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	mock_downloader "github.com/imager/mock/downloader"
	mock_model "github.com/imager/mock/model"
	mock_uploader "github.com/imager/mock/uploader"
	"github.com/imager/model"
)

const testFilePath = "./testdata/test.jpg"

func TestAll(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type tc struct {
		name               string
		getTest            func() *Service
		expectedStatusCode int
	}

	tcs := []tc{
		{
			name: "http.StatusOK",
			getTest: func() *Service {
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				ctx := context.Background()
				imagesSvc.EXPECT().All(ctx).Return([]model.OriginalResized{}, nil)
				return NewService(imagesSvc, nil, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "http.StatusInternalServerError",
			getTest: func() *Service {
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				ctx := context.Background()
				imagesSvc.EXPECT().All(ctx).Return(nil, errors.New("error"))
				return NewService(imagesSvc, nil, nil)
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			wr := httptest.NewRecorder()
			url, err := url.Parse("http://images")
			if err != nil {
				t.Fatal(err)
			}
			r := &http.Request{URL: url}
			tc.getTest().All(wr, r)
			statusCode := wr.Result().StatusCode
			if statusCode != tc.expectedStatusCode {
				t.Fatalf("expected status code is: %d but got: %d", tc.expectedStatusCode, statusCode)
			}
		})
	}
}

func createRecorderAndRequest(id string, w, h int) (*http.Request, *httptest.ResponseRecorder, error) {
	wr := httptest.NewRecorder()
	url, err := url.Parse(fmt.Sprintf("http://images?weight=%d&height=%d", w, h))
	if err != nil {
		return nil, nil, err
	}
	r := &http.Request{URL: url}
	r = mux.SetURLVars(r, map[string]string{"id": id})
	return r, wr, nil
}

func writeMultipartData(r *http.Request, b []byte) (*http.Request, error) {
	body := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(body)
	defer multipartWriter.Close()
	w, err := multipartWriter.CreateFormFile("file", "test.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	r, err = http.NewRequest("POST", r.URL.String(), body)
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", multipartWriter.FormDataContentType())
	return r, nil
}

func readImage() ([]byte, error) {
	return ioutil.ReadFile(testFilePath)
}

func resizeTestImage(w, h int, original []byte) (resizedImage []byte, err error) {
	img, err := imaging.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, err
	}
	img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
	buf := new(bytes.Buffer)
	if err := imaging.Encode(buf, img, imgFormat); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestResizeByID(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type tc struct {
		name               string
		getTest            func() (*Service, *http.Request, *httptest.ResponseRecorder)
		expectedStatusCode int
	}

	weight, height := 100, 100

	original, err := readImage()
	if err != nil {
		t.Fatal(err)
	}

	resized, err := resizeTestImage(weight, height, original)
	if err != nil {
		t.Fatal(err)
	}

	hash, err := calculateMD5(bytes.NewReader(resized))
	if err != nil {
		t.Fatal(err)
	}

	tcs := []tc{
		{
			name: "http.StatusBadRequest: invalid params",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", 0, 0)
				if err != nil {
					t.Fatal(err)
				}
				return NewService(nil, nil, nil), r, wr
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "http.StatusBadRequest: invalid id",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				return NewService(nil, nil, nil), r, wr
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "http.StatusInternalServerError: GetOne db error",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, errors.New("error"))
				return NewService(imagesSvc, nil, nil), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: Download error",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				downloadSvc.EXPECT().Download(r.Context(), "").Return(nil, errors.New("error"))
				return NewService(imagesSvc, nil, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: decoding file error",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				downloadSvc.EXPECT().Download(r.Context(), "").Return([]byte("test"), nil)
				return NewService(imagesSvc, nil, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: upload file error",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				downloadSvc.EXPECT().Download(r.Context(), "").Return(original, nil)
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hash), bytes.NewBuffer(resized)).Return("", errors.New("error"))
				return NewService(imagesSvc, uploadSvc, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: save file error",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				downloadSvc.EXPECT().Download(r.Context(), "").Return(original, nil)
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hash), bytes.NewBuffer(resized)).Return("", nil)
				imagesSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", weight, height)}).Return(0, errors.New("error"))
				return NewService(imagesSvc, uploadSvc, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusCreated",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				downloadSvc.EXPECT().Download(r.Context(), "").Return(original, nil)
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hash), bytes.NewBuffer(resized)).Return("", nil)
				imagesSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", weight, height)}).Return(1, nil)
				return NewService(imagesSvc, uploadSvc, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			svc, r, wr := tc.getTest()
			svc.ResizeByID(wr, r)
			statusCode := wr.Result().StatusCode
			if statusCode != tc.expectedStatusCode {
				t.Fatalf("expected status code is: %d but got: %d", tc.expectedStatusCode, statusCode)
			}
		})
	}
}

func originalImageSize(b []byte) (int, int, error) {
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return 0, 0, err
	}
	g := img.Bounds()
	return g.Dx(), g.Dy(), nil
}

func TestResize(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type tc struct {
		name               string
		getTest            func() (*Service, *http.Request, *httptest.ResponseRecorder)
		expectedStatusCode int
	}

	weight, height := 100, 100

	original, err := readImage()
	if err != nil {
		t.Fatal(err)
	}

	originalImageW, originalImageH, err := originalImageSize(original)
	if err != nil {
		t.Fatal(err)
	}

	resized, err := resizeTestImage(weight, height, original)
	if err != nil {
		t.Fatal(err)
	}

	hashOriginal, err := calculateMD5(bytes.NewBuffer(original))
	if err != nil {
		t.Fatal(err)
	}

	hashResized, err := calculateMD5(bytes.NewBuffer(resized))
	if err != nil {
		t.Fatal(err)
	}

	tcs := []tc{
		{
			name: "http.StatusBadRequest: invalid params",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", 0, 0)
				if err != nil {
					t.Fatal(err)
				}
				return NewService(nil, nil, nil), r, wr
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "http.StatusBadRequest: error decoding file",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", 100, 100)
				if err != nil {
					t.Fatal(err)
				}
				return NewService(nil, nil, nil), r, wr
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "http.StatusInternalServerError: error uploading original file",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				r, err = writeMultipartData(r, original)
				if err != nil {
					t.Fatal(err)
				}

				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashOriginal), bytes.NewBuffer(original)).Return("", errors.New("error"))
				uploadSvc.EXPECT().Upload(r.Context(), name(hashResized), bytes.NewBuffer(resized)).Return("", nil)
				return NewService(nil, uploadSvc, nil), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: error uploading resized file",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				r, err = writeMultipartData(r, original)
				if err != nil {
					t.Fatal(err)
				}
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashOriginal), bytes.NewBuffer(original)).Return("", nil)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashResized), bytes.NewBuffer(resized)).Return("", errors.New("error"))
				return NewService(nil, uploadSvc, nil), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: error saving original image",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				r, err = writeMultipartData(r, original)
				if err != nil {
					t.Fatal(err)
				}
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashOriginal), bytes.NewBuffer(original)).Return("", nil)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashResized), bytes.NewBuffer(resized)).Return("", nil)
				imageSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imageSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", originalImageW, originalImageH)}).Return(0, errors.New("error"))
				return NewService(imageSvc, uploadSvc, nil), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: error saving resized image",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				r, err = writeMultipartData(r, original)
				if err != nil {
					t.Fatal(err)
				}
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashOriginal), bytes.NewBuffer(original)).Return("", nil)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashResized), bytes.NewBuffer(resized)).Return("", nil)
				imageSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imageSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", originalImageW, originalImageH)}).Return(1, nil)
				imageSvc.EXPECT().Save(r.Context(), model.Image{OriginalID: 1, Resolution: fmt.Sprintf("%dx%d", weight, height)}).Return(0, errors.New("error"))
				return NewService(imageSvc, uploadSvc, nil), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusCreated",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				r, wr, err := createRecorderAndRequest("1", weight, height)
				if err != nil {
					t.Fatal(err)
				}
				r, err = writeMultipartData(r, original)
				if err != nil {
					t.Fatal(err)
				}
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashOriginal), bytes.NewBuffer(original)).Return("", nil)
				uploadSvc.EXPECT().Upload(r.Context(), name(hashResized), bytes.NewBuffer(resized)).Return("", nil)
				imageSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imageSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", originalImageW, originalImageH)}).Return(1, nil)
				imageSvc.EXPECT().Save(r.Context(), model.Image{OriginalID: 1, Resolution: fmt.Sprintf("%dx%d", weight, height)}).Return(2, nil)
				return NewService(imageSvc, uploadSvc, nil), r, wr
			},
			expectedStatusCode: http.StatusCreated,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			svc, r, wr := tc.getTest()
			svc.Resize(wr, r)
			statusCode := wr.Result().StatusCode
			if statusCode != tc.expectedStatusCode {
				t.Fatalf("expected status code is: %d but got: %d", tc.expectedStatusCode, statusCode)
			}
		})
	}
}

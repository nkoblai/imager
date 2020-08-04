package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
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

func resizeTestImage(w, h int) (originalImage []byte, resizedImage *bytes.Buffer, err error) {

	testFilePath := "./testdata/test.jpg"

	b, err := ioutil.ReadFile(testFilePath)
	if err != nil {
		return nil, nil, err
	}
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, nil, err
	}
	img = imaging.Resize(img, w, h, imaging.NearestNeighbor)
	buf := new(bytes.Buffer)
	if err := imaging.Encode(buf, img, imgFormat); err != nil {
		return nil, nil, err
	}
	return b, buf, nil
}
func TestResizeByID(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	type tc struct {
		name               string
		getTest            func() (*Service, *http.Request, *httptest.ResponseRecorder)
		expectedStatusCode int
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
				r, wr, err := createRecorderAndRequest("", 100, 100)
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
				r, wr, err := createRecorderAndRequest("1", 100, 100)
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
				r, wr, err := createRecorderAndRequest("1", 100, 100)
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
				r, wr, err := createRecorderAndRequest("1", 100, 100)
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
				w, h := 100, 100
				r, wr, err := createRecorderAndRequest("1", w, h)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				b, buf, err := resizeTestImage(w, h)
				if err != nil {
					t.Fatal(err)
				}
				hash, err := calculateMD5(bytes.NewReader(buf.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				downloadSvc.EXPECT().Download(r.Context(), "").Return(b, nil)
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hash), buf).Return("", errors.New("error"))
				return NewService(imagesSvc, uploadSvc, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusInternalServerError: save file error",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				w, h := 100, 100
				r, wr, err := createRecorderAndRequest("1", w, h)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				b, buf, err := resizeTestImage(w, h)
				if err != nil {
					t.Fatal(err)
				}
				hash, err := calculateMD5(bytes.NewReader(buf.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				downloadSvc.EXPECT().Download(r.Context(), "").Return(b, nil)
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hash), buf).Return("", nil)
				imagesSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", w, h)}).Return(0, errors.New("error"))
				return NewService(imagesSvc, uploadSvc, downloadSvc), r, wr
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "http.StatusCreated",
			getTest: func() (*Service, *http.Request, *httptest.ResponseRecorder) {
				w, h := 100, 100
				r, wr, err := createRecorderAndRequest("1", w, h)
				if err != nil {
					t.Fatal(err)
				}
				imagesSvc := mock_model.NewMockImagesRepository(mockCtrl)
				imagesSvc.EXPECT().GetOne(r.Context(), 1).Return(model.Image{}, nil)
				downloadSvc := mock_downloader.NewMockService(mockCtrl)
				b, buf, err := resizeTestImage(w, h)
				if err != nil {
					t.Fatal(err)
				}
				hash, err := calculateMD5(bytes.NewReader(buf.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				downloadSvc.EXPECT().Download(r.Context(), "").Return(b, nil)
				uploadSvc := mock_uploader.NewMockService(mockCtrl)
				uploadSvc.EXPECT().Upload(r.Context(), name(hash), buf).Return("", nil)
				imagesSvc.EXPECT().Save(r.Context(), model.Image{Resolution: fmt.Sprintf("%dx%d", w, h)}).Return(1, nil)
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

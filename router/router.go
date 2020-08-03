package router

import (
	"github.com/gorilla/mux"
	handler "github.com/imager/handler/v1/images"
	"github.com/imager/model"
	"github.com/imager/web/downloader"
	"github.com/imager/web/uploader"
)

func New(imgRepo model.ImagesRepository, uploadSvc uploader.Service, downloadSvc downloader.Service) *mux.Router {
	router := mux.NewRouter()
	imgSvcV1 := handler.NewService(imgRepo, uploadSvc, downloadSvc)

	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	apiV1.HandleFunc("/images", imgSvcV1.All).Methods("GET")
	apiV1.HandleFunc("/images", imgSvcV1.Resize).Methods("POST").Queries("height", "", "weight", "")
	apiV1.HandleFunc("/images/{id}", imgSvcV1.ResizeByID).Methods("POST").Queries("height", "", "weight", "")

	apiV1.HandleFunc("/images/resized", imgSvcV1.OnlyResized).Methods("GET")
	return router
}

package router

import (
	"github.com/gorilla/mux"
	"github.com/imager/handler/v1/images"
	"github.com/imager/model"
	"github.com/imager/uploader"
)

func New(imgRepo model.ImagesRepository, uploadSvc uploader.Service) *mux.Router {
	router := mux.NewRouter()
	imgSvcV1 := images.NewService(imgRepo, uploadSvc)

	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	apiV1.HandleFunc("/images", imgSvcV1.All).Methods("GET")
	apiV1.HandleFunc("/images", imgSvcV1.Resize).Methods("POST").Queries("height", "", "weight", "")

	apiV1.HandleFunc("/images/resized", imgSvcV1.OnlyResized).Methods("GET")
	return router
}

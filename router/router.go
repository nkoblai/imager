package router

import (
	"github.com/gorilla/mux"
	"github.com/imager/handler/v1/images"
	"github.com/imager/model"
)

func New(imgRepo model.ImagesRepository) *mux.Router {
	router := mux.NewRouter()
	imgSvcV1 := images.NewService(imgRepo)

	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	apiV1.HandleFunc("/images", imgSvcV1.All).Methods("GET")
	apiV1.HandleFunc("/images", imgSvcV1.Resize).Methods("POST")

	apiV1.HandleFunc("/images/resized", imgSvcV1.OnlyResized).Methods("GET")
	return router
}

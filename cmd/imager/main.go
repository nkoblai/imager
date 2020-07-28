package main

import (
	"database/sql"
	"net/http"

	"github.com/imager/repository/images"
	"github.com/imager/router"
)

func main() {
	// db, err := sql.Open("", "")
	// if err != nil {
	// 	log.Fatalf("error creating db connection: %v\n", err)
	// 	os.Exit(1)
	// }
	db := &sql.DB{}
	if err := http.ListenAndServe(":8080", router.New(images.NewRepo(db))); err != nil {
		panic(err)
	}
}
